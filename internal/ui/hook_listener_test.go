package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sjoeboo/hangar/internal/session"
)

func TestListenForHookChanges_NilWatcher(t *testing.T) {
	cmd := listenForHookChanges(nil)
	msg := cmd()
	if msg != nil {
		t.Errorf("nil watcher should return nil, got %T", msg)
	}
}

func TestListenForHookChanges_FiresAndReturnsMsg(t *testing.T) {
	watcher, err := session.NewStatusFileWatcher()
	if err != nil {
		t.Fatalf("NewStatusFileWatcher: %v", err)
	}

	// Kick the channel via the test helper before blocking on cmd()
	go watcher.TriggerForTest()

	cmd := listenForHookChanges(watcher)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		if _, ok := msg.(hookStatusChangedMsg); !ok {
			t.Errorf("expected hookStatusChangedMsg, got %T", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("listenForHookChanges did not fire within 1s")
	}
}
