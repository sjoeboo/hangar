//go:build !windows

package apiserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// handleSessionStream upgrades to a per-session WebSocket and streams raw PTY
// output from the tmux session to the browser in real time.
//
// The session is attached via:
//
//	tmux attach-session -f ignore-size -t <name>
//
// "ignore-size" prevents the web client's terminal dimensions from resizing the
// server-side tmux pane (which would affect the TUI).
//
// Protocol (same as agent-deck):
//   - Server → client: binary WebSocket frames containing raw PTY bytes
//   - Client → server: JSON text frames {"type":"input","data":"..."} or
//     {"type":"resize","cols":N,"rows":N}
func (s *APIServer) handleSessionStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	ts := inst.GetTmuxSession()
	if ts == nil {
		writeError(w, http.StatusBadRequest, "session has no tmux session")
		return
	}
	sessionName := ts.Name

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Debug("stream_ws_upgrade_failed", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()

	// Attach to the tmux session via PTY.
	// Without ignore-size, tmux uses this client's terminal dimensions for the
	// window size. The browser sends a resize message immediately after connecting
	// with the actual xterm.js cols/rows, so the session snaps to the browser
	// window size on connect and tracks it on every subsequent resize.
	cmd := exec.CommandContext(r.Context(), "tmux", "attach-session", "-t", sessionName)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		slog.Debug("stream_pty_start_failed",
			slog.String("session", sessionName),
			slog.String("error", err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to attach"}`))
		return
	}
	defer ptmx.Close()

	var writeMu sync.Mutex

	// PTY → WebSocket: stream raw bytes as binary frames.
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				writeMu.Lock()
				werr := conn.WriteMessage(websocket.BinaryMessage, chunk)
				writeMu.Unlock()
				if werr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					slog.Debug("stream_pty_read_error", slog.String("error", err.Error()))
				}
				return
			}
		}
	}()

	// WebSocket → PTY: forward input and handle resize commands.
	for {
		msgType, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		select {
		case <-done:
			return
		default:
		}
		if msgType == websocket.TextMessage {
			var msg struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				_, _ = ptmx.WriteString(msg.Data)
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols})
				}
			}
		}
	}

	<-done
}
