package ui

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StatusEvent represents a status change event from Claude hooks
type StatusEvent struct {
	InstanceID string `json:"instance_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Event      string `json:"event,omitempty"`
	Cwd        string `json:"cwd,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
}

// StatusWatcher monitors the status directory for Claude hook events
// This allows real-time status updates even when the TUI event loop is paused
// (e.g., when attached to a session via tea.Exec())
type StatusWatcher struct {
	watcher   *fsnotify.Watcher
	statusDir string
	eventCh   chan StatusEvent
	closeCh   chan struct{}

	// Debounce rapid events
	lastEvents   map[string]time.Time
	lastEventsMu sync.RWMutex
}

// NewStatusWatcher creates a watcher for Claude hook status events
func NewStatusWatcher(statusDir string) (*StatusWatcher, error) {
	// Create status directory if it doesn't exist
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return nil, err
	}

	// Create fsnotify watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the status directory
	if err := w.Add(statusDir); err != nil {
		w.Close()
		return nil, err
	}

	return &StatusWatcher{
		watcher:    w,
		statusDir:  statusDir,
		eventCh:    make(chan StatusEvent, 10), // Buffered to prevent blocking
		closeCh:    make(chan struct{}),
		lastEvents: make(map[string]time.Time),
	}, nil
}

// Start begins watching for status changes (non-blocking)
func (sw *StatusWatcher) Start() {
	go sw.watchLoop()
}

// watchLoop is the main event loop
func (sw *StatusWatcher) watchLoop() {
	for {
		select {
		case <-sw.closeCh:
			return

		case event, ok := <-sw.watcher.Events:
			if !ok {
				return
			}

			// Care about write and create events on .stop and .active files
			if event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0 {
				if strings.HasSuffix(event.Name, ".stop") || strings.HasSuffix(event.Name, ".active") {
					sw.processStatusFile(event.Name)
				}
			}

		case err, ok := <-sw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[STATUS-WATCHER] Error: %v", err)
		}
	}
}

// processStatusFile reads and processes a status file
func (sw *StatusWatcher) processStatusFile(filePath string) {
	// Debounce - ignore if we processed this file very recently
	sw.lastEventsMu.Lock()
	lastTime, exists := sw.lastEvents[filePath]
	if exists && time.Since(lastTime) < 500*time.Millisecond {
		sw.lastEventsMu.Unlock()
		return
	}
	sw.lastEvents[filePath] = time.Now()
	sw.lastEventsMu.Unlock()

	// Read the status file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[STATUS-WATCHER] Failed to read status file %s: %v", filePath, err)
		return
	}

	// Parse the JSON
	var evt StatusEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		log.Printf("[STATUS-WATCHER] Failed to parse status file %s: %v", filePath, err)
		return
	}

	log.Printf("[STATUS-WATCHER] Status event: instance=%s session=%s event=%s",
		evt.InstanceID, evt.SessionID, evt.Event)

	// Send to event channel (non-blocking)
	select {
	case sw.eventCh <- evt:
	default:
		log.Printf("[STATUS-WATCHER] Event channel full, dropping event for %s", evt.InstanceID)
	}

	// Clean up the status file after processing
	// This prevents stale files from accumulating
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay to ensure event is processed
		os.Remove(filePath)
	}()
}

// EventChannel returns the channel for receiving status events
func (sw *StatusWatcher) EventChannel() <-chan StatusEvent {
	return sw.eventCh
}

// Close stops the watcher
func (sw *StatusWatcher) Close() error {
	close(sw.closeCh)
	return sw.watcher.Close()
}

// StatusDir returns the default status directory path
func StatusDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/agent-deck-status"
	}
	return filepath.Join(home, ".agent-deck", "status")
}
