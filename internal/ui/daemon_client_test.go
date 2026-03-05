package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

func TestDaemonClient_TranslatesHookChanged(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	sentC := make(chan string, 4)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for msg := range sentC {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}))
	defer srv.Close()
	defer close(sentC)

	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	dc := &DaemonClient{msgC: make(chan daemonWSMsg, 8), done: make(chan struct{})}
	go dc.readLoop(conn)

	msg, _ := json.Marshal(map[string]string{"type": "hook_changed"})
	sentC <- string(msg)

	cmd := listenForDaemonEvents(dc)
	resultC := make(chan tea.Msg, 1)
	go func() { resultC <- cmd() }()

	select {
	case m := <-resultC:
		if _, ok := m.(hookStatusChangedMsg); !ok {
			t.Fatalf("expected hookStatusChangedMsg, got %T", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for hookStatusChangedMsg")
	}
}

func TestDaemonClient_TranslatesSessionsChanged(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	sentC := make(chan string, 4)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for msg := range sentC {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}))
	defer srv.Close()
	defer close(sentC)

	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	dc := &DaemonClient{msgC: make(chan daemonWSMsg, 8), done: make(chan struct{})}
	go dc.readLoop(conn)

	msg, _ := json.Marshal(map[string]string{"type": "sessions_changed"})
	sentC <- string(msg)

	cmd := listenForDaemonEvents(dc)
	resultC := make(chan tea.Msg, 1)
	go func() { resultC <- cmd() }()

	select {
	case m := <-resultC:
		if _, ok := m.(storageChangedMsg); !ok {
			t.Fatalf("expected storageChangedMsg, got %T", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for storageChangedMsg")
	}
}
