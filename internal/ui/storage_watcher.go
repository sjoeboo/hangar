package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ignoreWindow is the time window after NotifySave during which file changes are ignored.
// This prevents the watcher from triggering reload when the TUI itself saves.
const ignoreWindow = 500 * time.Millisecond

// StorageWatcher monitors sessions.json for external changes
type StorageWatcher struct {
	watcher     *fsnotify.Watcher
	storagePath string
	reloadCh    chan struct{}
	closeCh     chan struct{}

	// lastModified tracks file modification time for change detection
	lastModified time.Time
	modMu        sync.RWMutex

	// Tracks when TUI saved, to ignore self-triggered changes
	lastSaveTime time.Time
	saveMu       sync.RWMutex
}

// NewStorageWatcher creates a watcher for the given storage file
func NewStorageWatcher(storagePath string) (*StorageWatcher, error) {
	// Verify file exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("storage file does not exist: %s", storagePath)
	}

	// Create fsnotify watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	// Watch parent directory (handles atomic renames)
	dir := filepath.Dir(storagePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	// Get initial mod time
	info, _ := os.Stat(storagePath)
	lastMod := time.Time{}
	if info != nil {
		lastMod = info.ModTime()
	}

	return &StorageWatcher{
		watcher:      w,
		storagePath:  storagePath,
		lastModified: lastMod,
		reloadCh:     make(chan struct{}, 1), // Buffered to prevent blocking
		closeCh:      make(chan struct{}),
	}, nil
}

// Start begins watching for file changes (non-blocking)
func (sw *StorageWatcher) Start() {
	go sw.watchLoop()
}

// watchLoop is the main event loop (runs in goroutine)
func (sw *StorageWatcher) watchLoop() {
	debounce := time.NewTimer(0)
	debounce.Stop()

	for {
		select {
		case <-sw.closeCh:
			return

		case event, ok := <-sw.watcher.Events:
			if !ok {
				return
			}

			// Only care about our specific file
			if filepath.Base(event.Name) != filepath.Base(sw.storagePath) {
				continue
			}

			// Ignore if deleted (probably temp file)
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				continue
			}

			// Reset debounce timer (batches rapid writes)
			debounce.Reset(100 * time.Millisecond)

		case <-debounce.C:
			// Debounce period elapsed, check if file actually changed
			sw.checkAndNotify()

		case err, ok := <-sw.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			fmt.Fprintf(os.Stderr, "StorageWatcher error: %v\n", err)
		}
	}
}

// checkAndNotify checks file modification time and notifies if changed
func (sw *StorageWatcher) checkAndNotify() {
	// Check if we should ignore this change (TUI's own save)
	sw.saveMu.RLock()
	lastSave := sw.lastSaveTime
	sw.saveMu.RUnlock()

	if time.Since(lastSave) < ignoreWindow {
		// This change was likely caused by TUI's own save, ignore it
		// Still update lastModified to avoid re-triggering later
		if info, err := os.Stat(sw.storagePath); err == nil {
			sw.modMu.Lock()
			sw.lastModified = info.ModTime()
			sw.modMu.Unlock()
		}
		return
	}

	info, err := os.Stat(sw.storagePath)
	if err != nil {
		return // File might be temporarily gone during atomic rename
	}

	modTime := info.ModTime()
	sw.modMu.Lock()
	if modTime.After(sw.lastModified) {
		sw.lastModified = modTime
		sw.modMu.Unlock()

		// Non-blocking send (drop if channel full)
		select {
		case sw.reloadCh <- struct{}{}:
		default:
		}
	} else {
		sw.modMu.Unlock()
	}
}

// ReloadChannel returns the channel that signals when reload is needed
func (sw *StorageWatcher) ReloadChannel() <-chan struct{} {
	return sw.reloadCh
}

// NotifySave should be called by the TUI right before it saves to storage.
// This marks the current time so the watcher can ignore the resulting file change.
func (sw *StorageWatcher) NotifySave() {
	sw.saveMu.Lock()
	sw.lastSaveTime = time.Now()
	sw.saveMu.Unlock()
}

// Close stops the watcher and releases resources
func (sw *StorageWatcher) Close() error {
	close(sw.closeCh)
	return sw.watcher.Close()
}
