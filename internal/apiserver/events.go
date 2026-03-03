package apiserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPingPeriod = 30 * time.Second
	wsMaxMsgSize = 4096
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// Permissive origin check — Tailscale trust model.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents a single WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan WsMessage
}

// Hub manages all connected WebSocket clients.
// All client-map mutations happen inside run() to avoid races.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan WsMessage
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan WsMessage, 64),
		register:   make(chan *Client, 8),
		unregister: make(chan *Client, 8),
	}
}

// run is the Hub's event loop — must run in its own goroutine.
func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow client — drop and disconnect
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

// handleWS upgrades an HTTP connection to WebSocket and registers a new client.
func (s *APIServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Debug("ws_upgrade_failed", slog.String("error", err.Error()))
		return
	}

	c := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan WsMessage, 32),
	}
	s.hub.register <- c

	// Send hello immediately so the client knows it's connected.
	instances := s.instances()
	c.send <- WsMessage{
		Type: "hello",
		Data: WsHelloData{
			Version:  s.version,
			Sessions: len(instances),
		},
	}

	go c.writePump()
	go c.readPump(s)
}

// writePump drains c.send and writes to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			b, err := json.Marshal(msg)
			if err != nil {
				slog.Debug("ws_marshal_failed", slog.String("error", err.Error()))
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump reads commands from the client and dispatches them.
func (c *Client) readPump(s *APIServer) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(wsMaxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Debug("ws_read_error", slog.String("error", err.Error()))
			}
			return
		}

		var msg WsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			c.send <- WsMessage{Type: "pong"}
		case "send_message":
			s.wsHandleSendMessage(c, msg)
		case "stop_session":
			s.wsHandleStopSession(c, msg)
		}
	}
}
