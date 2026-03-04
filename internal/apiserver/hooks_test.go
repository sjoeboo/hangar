package apiserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjoeboo/hangar/internal/apiserver"
	"github.com/sjoeboo/hangar/internal/session"
)

func newTestWatcher(t *testing.T) *session.StatusFileWatcher {
	t.Helper()
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)
	t.Setenv("HOME", tmpDir)
	w, err := session.NewStatusFileWatcher()
	if err != nil {
		t.Fatalf("NewStatusFileWatcher: %v", err)
	}
	return w
}

func newTestServer(t *testing.T) *apiserver.APIServer {
	t.Helper()
	watcher := newTestWatcher(t)
	cfg := apiserver.APIConfig{Port: 0, BindAddress: "127.0.0.1"}
	return apiserver.New(cfg, watcher, nil, nil, nil, nil, "", "test")
}

func TestHookServer_ValidPayload(t *testing.T) {
	watcher := newTestWatcher(t)
	cfg := apiserver.APIConfig{Port: 0, BindAddress: "127.0.0.1"}
	srv := apiserver.New(cfg, watcher, nil, nil, nil, nil, "", "test")

	payload := map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hangar-Instance-Id", "inst-test-001")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	hs := watcher.GetHookStatus("inst-test-001")
	if hs == nil {
		t.Fatal("Expected hook status to be set")
	}
	if hs.Status != "running" {
		t.Errorf("Status = %q, want running", hs.Status)
	}
}

func TestHookServer_MissingInstanceID(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		"hook_event_name": "Stop",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even for missing instance ID", rr.Code)
	}
}

func TestHookServer_WrongMethod(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/hooks", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHookServer_UnknownEvent(t *testing.T) {
	watcher := newTestWatcher(t)
	cfg := apiserver.APIConfig{Port: 0, BindAddress: "127.0.0.1"}
	srv := apiserver.New(cfg, watcher, nil, nil, nil, nil, "", "test")

	payload := map[string]any{
		"hook_event_name": "SomeUnknownEvent",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-unknown")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if watcher.GetHookStatus("inst-unknown") != nil {
		t.Error("Unknown event should not set hook status")
	}
}

func TestHookServer_NotifiesWatcher(t *testing.T) {
	watcher := newTestWatcher(t)
	cfg := apiserver.APIConfig{Port: 0, BindAddress: "127.0.0.1"}
	srv := apiserver.New(cfg, watcher, nil, nil, nil, nil, "", "test")

	payload := map[string]any{
		"hook_event_name": "Stop",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-notify")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	select {
	case <-watcher.NotifyChannel():
		// correct
	default:
		t.Fatal("Expected notification on watcher channel after hook POST")
	}
}

func TestAPIServer_Status(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["version"]; !ok {
		t.Error("response missing 'version' field")
	}
}

func TestAPIServer_SessionsList_Empty(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var sessions []any
	if err := json.NewDecoder(rr.Body).Decode(&sessions); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestAPIServer_UIServesAssets(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	// The embedded assets directory exists (contains .gitkeep), so the file
	// server returns a 200 directory listing. Once the frontend build adds
	// index.html this will serve the SPA entry point instead.
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestAPIServer_CORS(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/sessions", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want 204", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header missing")
	}
}
