package apiserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sjoeboo/hangar/internal/session"
)

func newTestServerInternal(t *testing.T) *APIServer {
	t.Helper()
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)
	t.Setenv("HOME", tmpDir)
	watcher, err := session.NewStatusFileWatcher()
	if err != nil {
		t.Fatalf("NewStatusFileWatcher: %v", err)
	}
	return New(APIConfig{Port: 0}, watcher, nil, nil, nil, nil, "", "test")
}

func TestHookServer_BroadcastsHookChanged(t *testing.T) {
	srv := newTestServerInternal(t)

	// Run hub so broadcasts are processed.
	go srv.hub.run()

	// Register a fake client to receive broadcasts.
	c := &Client{
		hub:  srv.hub,
		conn: nil,
		send: make(chan WsMessage, 4),
	}
	srv.hub.register <- c
	// Give the hub goroutine a moment to process the registration.
	time.Sleep(10 * time.Millisecond)

	body := `{"hook_event_name":"UserPromptSubmit","session_id":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/hooks", strings.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-1")
	rr := httptest.NewRecorder()
	srv.handleHook(rr, req)

	select {
	case msg := <-c.send:
		if msg.Type != "hook_changed" {
			t.Fatalf("expected hook_changed, got %q", msg.Type)
		}
		data, ok := msg.Data.(WsHookChangedData)
		if !ok {
			t.Fatalf("expected WsHookChangedData, got %T", msg.Data)
		}
		if data.InstanceID != "inst-1" {
			t.Errorf("InstanceID = %q, want inst-1", data.InstanceID)
		}
		if data.HookEventName != "UserPromptSubmit" {
			t.Errorf("HookEventName = %q, want UserPromptSubmit", data.HookEventName)
		}
		if data.Status != "running" {
			t.Errorf("Status = %q, want running", data.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hook_changed broadcast")
	}
}
