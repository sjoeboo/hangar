// Package apiserver provides the embedded HTTP/WebSocket API server for Hangar.
// It exposes a full REST API for sessions, projects, and todos, plus a
// WebSocket endpoint for real-time push events.
//
// The server listens on the configured bind address (default: 0.0.0.0) and
// port (default: 47437), alongside the backward-compatible POST /hooks endpoint.
package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sjoeboo/hangar/internal/pr"
	"github.com/sjoeboo/hangar/internal/session"
	"github.com/sjoeboo/hangar/internal/webui"
)

// APIConfig holds the resolved configuration for the API server.
type APIConfig struct {
	Port        int
	BindAddress string
}

// APIServer is the embedded HTTP/WebSocket server.
// It serves the REST API, WebSocket endpoint, and the legacy /hooks endpoint.
type APIServer struct {
	cfg           APIConfig
	watcher       *session.StatusFileWatcher
	getInstances  func() []*session.Instance    // callback from TUI; caller holds instancesMu
	getPRInfo     func(sessionID string) *PRInfo // callback from TUI PR cache; may be nil
	triggerReload func()                         // callback to immediately trigger TUI DB reload
	prManager     *pr.Manager                   // unified PR data layer; may be nil in standalone mode
	profile       string
	hub           *Hub
	server        *http.Server
	startedAt     time.Time
	version       string
	done          chan struct{}
}

// New creates a new APIServer.
//
// getInstances is a callback that returns the current in-memory session slice.
// It must snapshot (copy) the slice while holding instancesMu. The callback is
// allowed to be nil (e.g. for tests), in which case session list endpoints
// return an empty list.
//
// getPRInfo is an optional callback that returns cached PR info for a session.
// Pass nil to omit PR data from session responses.
//
// triggerReload is an optional callback that immediately triggers the TUI to
// reload sessions from the DB (bypasses the 2-second poll interval). Pass nil
// if no TUI is running (e.g. tests).
//
// prManager is the unified PR data layer. Pass nil in standalone/test mode;
// when non-nil it is used as the authoritative PR source and background refresh
// is delegated to the manager (no internal PR refresh loop is started).
func New(cfg APIConfig, watcher *session.StatusFileWatcher, getInstances func() []*session.Instance, getPRInfo func(string) *PRInfo, triggerReload func(), prManager *pr.Manager, profile string, version string) *APIServer {
	hub := newHub()

	s := &APIServer{
		cfg:           cfg,
		watcher:       watcher,
		getInstances:  getInstances,
		getPRInfo:     getPRInfo,
		triggerReload: triggerReload,
		prManager:     prManager,
		profile:       profile,
		hub:           hub,
		startedAt:     time.Now(),
		version:       version,
		done:          make(chan struct{}),
	}

	// Register onChange callback on prManager so web clients receive PR updates.
	if prManager != nil {
		prManager.RegisterOnChange(func() {
			s.hub.broadcast <- WsMessage{Type: "sessions_changed"}
		})
	}

	mux := http.NewServeMux()

	// Backward-compatible hook receiver
	mux.HandleFunc("/hooks", s.handleHook)

	// REST API
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/sessions", s.handleSessions)
	mux.HandleFunc("/api/v1/sessions/{id}", s.handleSession)
	mux.HandleFunc("/api/v1/sessions/{id}/start", s.handleSessionStart)
	mux.HandleFunc("/api/v1/sessions/{id}/stop", s.handleSessionStop)
	mux.HandleFunc("/api/v1/sessions/{id}/restart", s.handleSessionRestart)
	mux.HandleFunc("/api/v1/sessions/{id}/send", s.handleSessionSend)
	mux.HandleFunc("/api/v1/sessions/{id}/output", s.handleSessionOutput)
	mux.HandleFunc("/api/v1/sessions/{id}/stream", s.handleSessionStream)
	mux.HandleFunc("/api/v1/projects", s.handleProjects)
	mux.HandleFunc("/api/v1/projects/{id}", s.handleProject)
	mux.HandleFunc("/api/v1/todos", s.handleTodos)
	mux.HandleFunc("/api/v1/todos/{id}", s.handleTodo)

	// PR dashboard endpoints (require prManager)
	mux.HandleFunc("/api/v1/prs", s.handlePRDashboard)
	mux.HandleFunc("/api/v1/prs/detail", s.handlePRDetail)
	mux.HandleFunc("/api/v1/prs/review", s.handlePRReview)
	mux.HandleFunc("/api/v1/prs/comment", s.handlePRComment)
	mux.HandleFunc("/api/v1/prs/state", s.handlePRState)

	// WebSocket
	mux.HandleFunc("/api/v1/ws", s.handleWS)

	// Serve embedded web UI assets; fall back to index.html for SPA routing
	uiFS := webui.Assets()
	uiHandler := http.FileServer(uiFS)
	mux.HandleFunc("/ui/", func(w http.ResponseWriter, r *http.Request) {
		// Strip /ui prefix so the file server sees paths relative to dist/
		path := strings.TrimPrefix(r.URL.Path, "/ui")
		if path == "" {
			path = "/"
		}
		// Check if the file exists in the embedded FS; serve index.html for unknown paths (SPA routing)
		f, err := uiFS.Open(path)
		if err != nil {
			// SPA fallback: serve index.html for client-side routes
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/ui/index.html"
			http.StripPrefix("/ui", uiHandler).ServeHTTP(w, r2)
			return
		}
		f.Close()
		http.StripPrefix("/ui", uiHandler).ServeHTTP(w, r)
	})

	s.server = &http.Server{
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s
}

// ServeHTTP implements http.Handler — delegates to the internal mux.
// Used by tests to call handlers directly without starting a listener.
func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.Handler.ServeHTTP(w, r)
}

// Start binds to the configured address and begins serving.
// It also starts the WebSocket hub and wires status watcher notifications to hub broadcasts.
// Blocks until ctx is cancelled; returns nil on clean shutdown.
func (s *APIServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.BindAddress, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("apiserver listen %s: %w", addr, err)
	}
	slog.Info("apiserver_started", slog.String("addr", addr))

	// Run WebSocket hub
	go s.hub.run()

	// Bridge status watcher notifications → WS broadcasts
	if s.watcher != nil {
		go s.bridgeWatcherToHub(ctx)
	}

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
	case err := <-errCh:
		close(s.done)
		return err
	}
	close(s.done)
	return nil
}

// WaitDone returns a channel closed when Start() has returned.
func (s *APIServer) WaitDone() <-chan struct{} {
	return s.done
}

// bridgeWatcherToHub listens for hook notifications and broadcasts session_updated events.
func (s *APIServer) bridgeWatcherToHub(ctx context.Context) {
	ch := s.watcher.NotifyChannel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			// Broadcast the full session list refresh signal; clients can re-fetch
			// or we can send individual session updates. For now send a lightweight ping.
			s.hub.broadcast <- WsMessage{Type: "sessions_changed"}
			// TODO: broadcast session_output events here when per-session change
			// notifications are available. Clients use GET /api/v1/sessions/:id/output
			// for the initial terminal snapshot.
		}
	}
}

// corsMiddleware adds permissive CORS headers — appropriate for Tailscale trust model.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Hangar-Instance-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleStatus serves GET /api/v1/status.
func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	instances := s.instances()
	byStatus := make(map[string]int)
	for _, inst := range instances {
		byStatus[string(inst.Status)]++
	}

	resp := StatusResponse{
		Version:  s.version,
		Uptime:   time.Since(s.startedAt).Round(time.Second).String(),
		Sessions: len(instances),
		ByStatus: byStatus,
	}
	writeJSON(w, http.StatusOK, resp)
}

// instances returns a snapshot of the current session list.
// Returns an empty slice if the callback is nil.
func (s *APIServer) instances() []*session.Instance {
	if s.getInstances == nil {
		return nil
	}
	return s.getInstances()
}

// writeJSON marshals v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response: {"error": "..."}.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
