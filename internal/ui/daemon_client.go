package ui

import (
	"encoding/json"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// daemonWSMsg is a parsed event received from the daemon WebSocket.
type daemonWSMsg struct {
	eventType string
}

// DaemonClient maintains a WebSocket connection to a running hangar daemon
// and surfaces events as Bubble Tea commands.
// Re-issue listenForDaemonEvents after each message to keep receiving —
// exactly like listenForHookChanges.
type DaemonClient struct {
	url  string
	msgC chan daemonWSMsg
	done chan struct{}
}

// newDaemonClient dials the daemon WebSocket at the given port and starts
// a background read loop. Returns nil if the connection fails.
func newDaemonClient(port int) *DaemonClient {
	url := fmt.Sprintf("ws://127.0.0.1:%d/api/v1/ws", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		slog.Warn("daemon_ws_connect_failed", slog.String("url", url), slog.String("error", err.Error()))
		return nil
	}

	dc := &DaemonClient{
		url:  url,
		msgC: make(chan daemonWSMsg, 32),
		done: make(chan struct{}),
	}
	go dc.readLoop(conn)
	return dc
}

func (dc *DaemonClient) readLoop(conn *websocket.Conn) {
	defer close(dc.done)
	defer conn.Close()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("daemon_ws_read_error", slog.String("error", err.Error()))
			return
		}

		var msg struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}

		switch msg.Type {
		case "hook_changed", "sessions_changed", "session_created", "session_deleted":
			select {
			case dc.msgC <- daemonWSMsg{eventType: msg.Type}:
			default:
				// Buffer full — TUI will reload on the next tick anyway
			}
		}
	}
}

// Close stops the DaemonClient. The underlying connection will be closed
// when readLoop exits (either due to an error or the remote end closing).
// Safe to call on a nil receiver.
func (dc *DaemonClient) Close() {
	if dc == nil {
		return
	}
	// We don't store the conn directly; closing it is handled by readLoop.
	// The caller should cancel the context that owns this client.
}

// listenForDaemonEvents returns a tea.Cmd that blocks until the next WS
// event arrives from the daemon, then returns the appropriate Bubble Tea
// message. MUST be re-issued in Update after each message to keep listening.
func listenForDaemonEvents(dc *DaemonClient) tea.Cmd {
	return func() tea.Msg {
		if dc == nil {
			return nil
		}
		select {
		case msg, ok := <-dc.msgC:
			if !ok {
				return nil
			}
			switch msg.eventType {
			case "hook_changed":
				return hookStatusChangedMsg{}
			case "sessions_changed", "session_created", "session_deleted":
				return storageChangedMsg{}
			}
		case <-dc.done:
			return nil
		}
		return nil
	}
}
