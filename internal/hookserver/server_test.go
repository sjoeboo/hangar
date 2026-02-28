package hookserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sjoeboo/hangar/internal/hookserver"
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

func TestHookServer_ValidPayload(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

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

	time.Sleep(10 * time.Millisecond)
	hs := watcher.GetHookStatus("inst-test-001")
	if hs == nil {
		t.Fatal("Expected hook status to be set")
	}
	if hs.Status != "running" {
		t.Errorf("Status = %q, want running", hs.Status)
	}
}

func TestHookServer_MissingInstanceID(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

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
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

	req := httptest.NewRequest(http.MethodGet, "/hooks", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHookServer_UnknownEvent(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

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
	srv := hookserver.New(0, watcher)

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
