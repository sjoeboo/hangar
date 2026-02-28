package hookserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/sjoeboo/hangar/internal/session"
)

// HookServer is an embedded HTTP server that receives Claude Code webhook events.
// It binds to 127.0.0.1 only and is lifecycle-bound to the TUI process.
type HookServer struct {
	port    int
	watcher *session.StatusFileWatcher
	server  *http.Server
}

// New creates a new HookServer. port=0 is valid for tests (use ServeHTTP directly).
func New(port int, watcher *session.StatusFileWatcher) *HookServer {
	s := &HookServer{
		port:    port,
		watcher: watcher,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/hooks", s.handleHook)
	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

// ServeHTTP implements http.Handler for testing — delegates directly to the mux.
func (s *HookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.Handler.ServeHTTP(w, r)
}

// Start binds to 127.0.0.1:{port} and begins serving. Blocks until ctx is cancelled.
// Returns nil on clean shutdown.
func (s *HookServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return fmt.Errorf("hookserver listen :%d: %w", s.port, err)
	}
	slog.Info("hookserver_started", slog.Int("port", s.port))

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

// hookPayload is the JSON body Claude Code sends for HTTP hook events.
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Matcher       string `json:"matcher,omitempty"`
}

func (s *HookServer) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	instanceID := r.Header.Get("X-Hangar-Instance-Id")
	if instanceID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64KB max
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload hookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := session.MapEventToStatus(payload.HookEventName)

	// Notification events: only "waiting" for permission_prompt/elicitation_dialog
	if payload.HookEventName == "Notification" {
		if payload.Matcher == "permission_prompt" || payload.Matcher == "elicitation_dialog" {
			status = "waiting"
		}
	}

	if status != "" {
		s.watcher.Notify(instanceID, status, payload.SessionID, payload.HookEventName)
	}

	w.WriteHeader(http.StatusOK)
}
