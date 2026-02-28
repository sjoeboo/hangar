package session

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/sjoeboo/hangar/internal/logging"
)

var hookLog = logging.ForComponent(logging.CompSession)

// HookStatus holds the decoded status from a hook status file.
type HookStatus struct {
	Status    string    // running, idle, waiting, dead
	SessionID string    // Claude session ID
	Event     string    // Hook event name
	UpdatedAt time.Time // When this status was received
}

// StatusFileWatcher watches ~/.hangar/hooks/ for status file changes
// and updates instance hook status in real time.
type StatusFileWatcher struct {
	hooksDir string
	watcher  *fsnotify.Watcher

	mu       sync.RWMutex
	statuses map[string]*HookStatus // instance_id -> latest hook status

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once // ensures hookChangedCh is closed exactly once

	// sendMu serialises channel sends (processFile) against the channel close
	// (Stop) to prevent a concurrent send-on-closed-channel panic/race.
	sendMu sync.Mutex

	// hookChangedCh is sent to (non-blocking) when a hook status file is processed.
	hookChangedCh chan struct{}
}

// NewStatusFileWatcher creates a new watcher for the hooks directory.
// Call Start() to begin watching.
func NewStatusFileWatcher() (*StatusFileWatcher, error) {
	hooksDir := GetHooksDir()

	// Ensure directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &StatusFileWatcher{
		hooksDir:      hooksDir,
		watcher:       watcher,
		statuses:      make(map[string]*HookStatus),
		ctx:           ctx,
		cancel:        cancel,
		hookChangedCh: make(chan struct{}, 1),
	}, nil
}

// NotifyChannel returns a receive-only channel that fires when a hook status
// file is successfully parsed and the in-memory status map is updated.
// The channel has capacity 1 — rapid successive file changes coalesce into a
// single signal. Consumers should do a full status reload on each signal
// rather than assuming a 1:1 mapping to file changes.
//
// Notifications are NOT sent when a file cannot be read or parsed; such errors
// are silently dropped. The channel is closed when Stop() is called; consumers
// must use the two-value receive form and return on ok==false.
func (w *StatusFileWatcher) NotifyChannel() <-chan struct{} {
	return w.hookChangedCh
}

// Start begins watching the hooks directory. Must be called in a goroutine.
// On startup, loadExisting() processes all pre-existing files; these may
// coalesce into a single channel notification if multiple files are present.
func (w *StatusFileWatcher) Start() {
	if err := w.watcher.Add(w.hooksDir); err != nil {
		hookLog.Warn("hook_watcher_add_failed", slog.String("dir", w.hooksDir), slog.String("error", err.Error()))
		return
	}

	// Load any existing status files on startup
	w.loadExisting()

	// Debounce timer: coalesce rapid file events
	var debounceTimer *time.Timer
	defer func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
	}()
	pendingFiles := make(map[string]bool)
	var pendingMu sync.Mutex

	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only process .json file writes/creates
			if filepath.Ext(event.Name) != ".json" {
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}

			pendingMu.Lock()
			pendingFiles[event.Name] = true
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
				pendingMu.Lock()
				files := make([]string, 0, len(pendingFiles))
				for f := range pendingFiles {
					files = append(files, f)
				}
				pendingFiles = make(map[string]bool)
				pendingMu.Unlock()

				for _, f := range files {
					w.processFile(f)
				}
			})
			pendingMu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			hookLog.Warn("hook_watcher_error", slog.String("error", err.Error()))
		}
	}
}

// Stop shuts down the watcher. Safe to call multiple times.
func (w *StatusFileWatcher) Stop() {
	w.cancel()
	_ = w.watcher.Close()
	w.stopOnce.Do(func() {
		// Hold sendMu while closing so processFile cannot send concurrently.
		w.sendMu.Lock()
		close(w.hookChangedCh) // unblock any goroutine blocked on NotifyChannel()
		w.sendMu.Unlock()
	})
}

// TriggerForTest sends a notification to the channel for testing purposes.
// Do not call from production code.
func (w *StatusFileWatcher) TriggerForTest() {
	select {
	case w.hookChangedCh <- struct{}{}:
	default:
	}
}

// GetHookStatus returns the hook status for an instance, or nil if not available.
func (w *StatusFileWatcher) GetHookStatus(instanceID string) *HookStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.statuses[instanceID]
}

// loadExisting reads all current status files on startup.
func (w *StatusFileWatcher) loadExisting() {
	entries, err := os.ReadDir(w.hooksDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		w.processFile(filepath.Join(w.hooksDir, entry.Name()))
	}
}

// processFile reads a status file and updates the internal map.
func (w *StatusFileWatcher) processFile(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var status struct {
		Status    string `json:"status"`
		SessionID string `json:"session_id"`
		Event     string `json:"event"`
		Timestamp int64  `json:"ts"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return
	}

	// Extract instance ID from filename (remove .json extension)
	base := filepath.Base(filePath)
	instanceID := strings.TrimSuffix(base, ".json")

	hookStatus := &HookStatus{
		Status:    status.Status,
		SessionID: status.SessionID,
		Event:     status.Event,
		UpdatedAt: time.Unix(status.Timestamp, 0),
	}

	w.mu.Lock()
	w.statuses[instanceID] = hookStatus
	w.mu.Unlock()

	hookLog.Debug("hook_status_updated",
		slog.String("instance", instanceID),
		slog.String("status", status.Status),
		slog.String("event", status.Event),
	)

	// Serialise against Stop() which closes hookChangedCh under sendMu.
	// Holding sendMu here prevents a concurrent send-on-closed-channel panic/race.
	if w.hookChangedCh != nil {
		w.sendMu.Lock()
		// Re-check context inside the lock: Stop() calls cancel() before closing
		// the channel, so ctx.Done() being set means the channel may already be
		// closed (or about to be closed).
		cancelled := w.ctx != nil && w.ctx.Err() != nil
		if !cancelled {
			select {
			case w.hookChangedCh <- struct{}{}:
			default: // already pending, coalesce
			}
		}
		w.sendMu.Unlock()
	}
}

// GetHooksDir returns the path to the hooks status directory.
func GetHooksDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".hangar", "hooks")
	}
	return filepath.Join(home, ".hangar", "hooks")
}
