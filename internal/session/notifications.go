package session

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// NotificationEntry represents a waiting session in the notification bar
type NotificationEntry struct {
	SessionID    string
	TmuxName     string
	Title        string
	AssignedKey  string
	WaitingSince time.Time
}

// NotificationManager tracks waiting sessions for the notification bar
type NotificationManager struct {
	entries  []*NotificationEntry // Ordered: newest first
	maxShown int
	mu       sync.RWMutex
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(maxShown int) *NotificationManager {
	if maxShown <= 0 {
		maxShown = 6
	}
	return &NotificationManager{
		entries:  make([]*NotificationEntry, 0),
		maxShown: maxShown,
	}
}

// Add registers a session as waiting (newest goes to position [0])
func (nm *NotificationManager) Add(inst *Instance) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Already tracked?
	for _, e := range nm.entries {
		if e.SessionID == inst.ID {
			return nil
		}
	}

	// Create entry
	tmuxName := ""
	if ts := inst.GetTmuxSession(); ts != nil {
		tmuxName = ts.Name
	}
	entry := &NotificationEntry{
		SessionID:    inst.ID,
		TmuxName:     tmuxName,
		Title:        inst.Title,
		WaitingSince: time.Now(),
	}

	// Prepend (newest first)
	nm.entries = append([]*NotificationEntry{entry}, nm.entries...)

	// Trim to max
	if len(nm.entries) > nm.maxShown {
		nm.entries = nm.entries[:nm.maxShown]
	}

	// Reassign keys (1, 2, 3, ...)
	nm.reassignKeys()

	return nil
}

// Remove removes a session from notifications
func (nm *NotificationManager) Remove(sessionID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i, e := range nm.entries {
		if e.SessionID == sessionID {
			nm.entries = append(nm.entries[:i], nm.entries[i+1:]...)
			break
		}
	}

	// Reassign keys
	nm.reassignKeys()
}

// reassignKeys assigns keys 1-6 based on position
func (nm *NotificationManager) reassignKeys() {
	for i, e := range nm.entries {
		e.AssignedKey = fmt.Sprintf("%d", i+1)
	}
}

// GetEntries returns a copy of current entries (newest first)
func (nm *NotificationManager) GetEntries() []*NotificationEntry {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	result := make([]*NotificationEntry, len(nm.entries))
	copy(result, nm.entries)
	return result
}

// GetSessionByKey returns the entry for a given key (1-6)
func (nm *NotificationManager) GetSessionByKey(key string) *NotificationEntry {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, e := range nm.entries {
		if e.AssignedKey == key {
			return e
		}
	}
	return nil
}

// Count returns number of notifications
func (nm *NotificationManager) Count() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return len(nm.entries)
}

// Clear removes all notifications
func (nm *NotificationManager) Clear() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.entries = make([]*NotificationEntry, 0)
}

// Has checks if a session is in notifications
func (nm *NotificationManager) Has(sessionID string) bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, e := range nm.entries {
		if e.SessionID == sessionID {
			return true
		}
	}
	return false
}

// FormatBar returns the formatted status bar text
func (nm *NotificationManager) FormatBar() string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if len(nm.entries) == 0 {
		return ""
	}

	var parts []string
	for _, e := range nm.entries {
		// No truncation - show full title, tmux will handle overflow
		parts = append(parts, fmt.Sprintf("[%s] %s", e.AssignedKey, e.Title))
	}

	return "⚡ " + strings.Join(parts, " ")
}

// SyncFromInstances updates notifications based on current instance states
// Call this periodically to sync with actual session statuses
func (nm *NotificationManager) SyncFromInstances(instances []*Instance, currentSessionID string) (added, removed []string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Build set of currently waiting sessions (excluding current)
	waitingSet := make(map[string]*Instance)
	for _, inst := range instances {
		if inst.Status == StatusWaiting && inst.ID != currentSessionID {
			waitingSet[inst.ID] = inst
		}
	}

	// Remove entries that are no longer waiting
	newEntries := make([]*NotificationEntry, 0)
	for _, e := range nm.entries {
		if _, stillWaiting := waitingSet[e.SessionID]; stillWaiting {
			newEntries = append(newEntries, e)
			delete(waitingSet, e.SessionID) // Don't re-add
		} else {
			removed = append(removed, e.SessionID)
		}
	}
	nm.entries = newEntries

	// Add new waiting sessions to entries
	for _, inst := range waitingSet {
		tmuxName := ""
		if ts := inst.GetTmuxSession(); ts != nil {
			tmuxName = ts.Name
		}
		entry := &NotificationEntry{
			SessionID:    inst.ID,
			TmuxName:     tmuxName,
			Title:        inst.Title,
			WaitingSince: inst.GetWaitingSince(),
		}
		nm.entries = append(nm.entries, entry)
		added = append(added, inst.ID)
	}

	// Sort ALL entries by WaitingSince (newest first)
	// This ensures correct ordering regardless of how entries were added
	sort.Slice(nm.entries, func(i, j int) bool {
		return nm.entries[i].WaitingSince.After(nm.entries[j].WaitingSince)
	})

	// Trim to maxShown (keeps the newest waiting sessions)
	if len(nm.entries) > nm.maxShown {
		nm.entries = nm.entries[:nm.maxShown]
	}

	// Reassign keys
	nm.reassignKeys()

	return added, removed
}
