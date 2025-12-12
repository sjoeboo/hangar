package ui

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
	"github.com/asheshgoplani/agent-deck/internal/update"
)

// Version is set by main.go for update checking
var Version = "0.0.0"

// SetVersion sets the current version for update checking
func SetVersion(v string) {
	Version = v
}

// Terminal escape sequences for smooth transitions
const (
	// Synchronized output (DEC mode 2026) - batches screen updates for atomic rendering
	// Supported by iTerm2, kitty, Alacritty, WezTerm, and other modern terminals
	syncOutputBegin = "\x1b[?2026h"
	syncOutputEnd   = "\x1b[?2026l"

	// Screen clear + cursor home
	clearScreen = "\033[2J\033[H"

	// tickInterval for UI refresh - event-driven detection still reduces
	// expensive operations (SignalFileActivity updates state, tick just redraws)
	tickInterval = 500 * time.Millisecond
)

// Home is the main application model
type Home struct {
	// Dimensions
	width  int
	height int

	// Profile
	profile string // The profile this Home is displaying

	// Data (protected by instancesMu for background worker access)
	instances   []*session.Instance
	instancesMu sync.RWMutex // Protects instances slice for thread-safe background access
	storage     *session.Storage
	groupTree   *session.GroupTree
	flatItems   []session.Item // Flattened view for cursor navigation

	// Components
	search        *Search
	newDialog     *NewDialog
	groupDialog   *GroupDialog   // For creating/renaming groups
	forkDialog    *ForkDialog    // For forking sessions
	confirmDialog *ConfirmDialog // For confirming destructive actions
	helpOverlay   *HelpOverlay   // For showing keyboard shortcuts

	// State
	cursor      int  // Selected item index in flatItems
	viewOffset  int  // First visible item index (for scrolling)
	isAttaching bool // Prevents View() output during attach (fixes Bubble Tea Issue #431)
	err         error

	// Preview cache (async fetching - View() must be pure, no blocking I/O)
	previewCache      map[string]string // sessionID -> cached preview content
	previewCacheMu    sync.RWMutex      // Protects previewCache for thread-safety
	previewFetchingID string            // ID currently being fetched (prevents duplicate fetches)

	// Round-robin status updates (Priority 1A optimization)
	// Instead of updating ALL sessions every tick, we update batches of 5-10 sessions
	// This reduces CPU usage by 90%+ while maintaining responsiveness
	statusUpdateIndex atomic.Int32 // Current position in round-robin cycle (atomic for thread safety)

	// Background status worker (Priority 1C optimization)
	// Moves status updates to a separate goroutine, completely decoupling from UI
	statusTrigger    chan statusUpdateRequest // Triggers background status update
	statusWorkerDone chan struct{}            // Signals worker has stopped

	// Event-driven status detection (Priority 2)
	logWatcher *tmux.LogWatcher

	// Storage warning (shown if storage initialization failed)
	storageWarning string

	// Update notification (async check on startup)
	updateInfo *update.UpdateInfo

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

// Messages
type loadSessionsMsg struct {
	instances []*session.Instance
	groups    []*session.GroupData
	err       error
}

type sessionCreatedMsg struct {
	instance *session.Instance
	err      error
}

type sessionForkedMsg struct {
	instance *session.Instance
	err      error
}

type refreshMsg struct{}

type statusUpdateMsg struct{} // Triggers immediate status update without reloading

type updateCheckMsg struct {
	info *update.UpdateInfo
}

type tickMsg time.Time

// previewFetchedMsg is sent when async preview content is ready
type previewFetchedMsg struct {
	sessionID string
	content   string
	err       error
}

// statusUpdateRequest is sent to the background worker with current viewport info
type statusUpdateRequest struct {
	viewOffset    int   // Current scroll position
	visibleHeight int   // How many items fit on screen
	flatItemIDs   []string // IDs of sessions in current flatItems order (for visible detection)
}

// NewHome creates a new home model with the default profile
func NewHome() *Home {
	return NewHomeWithProfile("")
}

// NewHomeWithProfile creates a new home model with the specified profile
func NewHomeWithProfile(profile string) *Home {
	ctx, cancel := context.WithCancel(context.Background())

	var storageWarning string
	storage, err := session.NewStorageWithProfile(profile)
	if err != nil {
		// Log the error and set warning - sessions won't persist but app will still function
		log.Printf("Warning: failed to initialize storage, sessions won't persist: %v", err)
		storageWarning = fmt.Sprintf("⚠ Storage unavailable: %v (sessions won't persist)", err)
		storage = nil
	}

	// Get the actual profile name (could be resolved from env var or config)
	actualProfile := session.DefaultProfile
	if storage != nil {
		actualProfile = storage.Profile()
	}

	h := &Home{
		profile:          actualProfile,
		storage:          storage,
		storageWarning:   storageWarning,
		search:           NewSearch(),
		newDialog:        NewNewDialog(),
		groupDialog:      NewGroupDialog(),
		forkDialog:       NewForkDialog(),
		confirmDialog:    NewConfirmDialog(),
		helpOverlay:      NewHelpOverlay(),
		cursor:           0,
		ctx:              ctx,
		cancel:           cancel,
		instances:        []*session.Instance{},
		groupTree:        session.NewGroupTree([]*session.Instance{}),
		flatItems:        []session.Item{},
		previewCache:     make(map[string]string),
		statusTrigger:    make(chan statusUpdateRequest, 1), // Buffered to avoid blocking
		statusWorkerDone: make(chan struct{}),
	}

	// Initialize event-driven log watcher
	logWatcher, err := tmux.NewLogWatcher(tmux.LogDir(), func(sessionName string) {
		// Find session by tmux name and signal file activity
		h.instancesMu.RLock()
		for _, inst := range h.instances {
			if inst.GetTmuxSession() != nil && inst.GetTmuxSession().Name == sessionName {
				// Signal file activity (triggers GREEN) then update status
				go func(i *session.Instance) {
					if tmuxSess := i.GetTmuxSession(); tmuxSess != nil {
						tmuxSess.SignalFileActivity() // Directly triggers GREEN
					}
					_ = i.UpdateStatus()
				}(inst)
				break
			}
		}
		h.instancesMu.RUnlock()
	})
	if err != nil {
		log.Printf("Warning: failed to create log watcher: %v (falling back to polling)", err)
	} else {
		h.logWatcher = logWatcher
		go h.logWatcher.Start()
	}

	// Start background status worker (Priority 1C)
	go h.statusWorker()

	return h
}

// rebuildFlatItems rebuilds the flattened view from group tree
func (h *Home) rebuildFlatItems() {
	h.flatItems = h.groupTree.Flatten()
	// Ensure cursor is valid
	if h.cursor >= len(h.flatItems) {
		h.cursor = len(h.flatItems) - 1
	}
	if h.cursor < 0 {
		h.cursor = 0
	}
	// Adjust viewport if cursor is out of view
	h.syncViewport()
}

// syncViewport ensures the cursor is visible within the viewport
// Call this after any cursor movement
func (h *Home) syncViewport() {
	if len(h.flatItems) == 0 {
		h.viewOffset = 0
		return
	}

	// Calculate visible height for session list
	// Header takes 2 lines, help bar takes 3 lines, content area needs -2 for title
	helpBarHeight := 3
	contentHeight := h.height - 2 - helpBarHeight
	visibleHeight := contentHeight - 2 // -2 for SESSIONS title
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// If cursor is above viewport, scroll up
	if h.cursor < h.viewOffset {
		h.viewOffset = h.cursor
	}

	// If cursor is below viewport, scroll down
	// Leave room for "⋮ +N more" indicator
	maxVisible := visibleHeight - 1
	if maxVisible < 1 {
		maxVisible = 1
	}
	if h.cursor >= h.viewOffset+maxVisible {
		h.viewOffset = h.cursor - maxVisible + 1
	}

	// Clamp viewOffset to valid range
	maxOffset := len(h.flatItems) - maxVisible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if h.viewOffset > maxOffset {
		h.viewOffset = maxOffset
	}
	if h.viewOffset < 0 {
		h.viewOffset = 0
	}
}

// jumpToRootGroup jumps the cursor to the Nth root-level group (1-indexed)
// Root groups are those at Level 0 (no "/" in path)
func (h *Home) jumpToRootGroup(n int) {
	if n < 1 || n > 9 {
		return
	}

	// Find the Nth root group in flatItems
	rootGroupCount := 0
	for i, item := range h.flatItems {
		if item.Type == session.ItemTypeGroup && item.Level == 0 {
			rootGroupCount++
			if rootGroupCount == n {
				h.cursor = i
				h.syncViewport()
				return
			}
		}
	}
	// If n exceeds available root groups, do nothing (no-op)
}

// Init initializes the model
func (h *Home) Init() tea.Cmd {
	return tea.Batch(
		h.loadSessions,
		h.tick(),
		h.checkForUpdate(),
	)
}

// checkForUpdate checks for updates asynchronously
func (h *Home) checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		info, _ := update.CheckForUpdate(Version, false)
		return updateCheckMsg{info: info}
	}
}

// loadSessions loads sessions from storage
func (h *Home) loadSessions() tea.Msg {
	if h.storage == nil {
		return loadSessionsMsg{instances: []*session.Instance{}, err: fmt.Errorf("storage not initialized")}
	}

	instances, groups, err := h.storage.LoadWithGroups()
	return loadSessionsMsg{instances: instances, groups: groups, err: err}
}

// tick returns a command that sends a tick message at regular intervals
// Status updates use time-based cooldown to prevent flickering
func (h *Home) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchPreview returns a command that asynchronously fetches preview content
// This keeps View() pure (no blocking I/O) as per Bubble Tea best practices
func (h *Home) fetchPreview(inst *session.Instance) tea.Cmd {
	if inst == nil {
		return nil
	}
	sessionID := inst.ID
	return func() tea.Msg {
		content, err := inst.PreviewFull()
		return previewFetchedMsg{
			sessionID: sessionID,
			content:   content,
			err:       err,
		}
	}
}

// getSelectedSession returns the currently selected session, or nil if a group is selected
func (h *Home) getSelectedSession() *session.Instance {
	if len(h.flatItems) == 0 || h.cursor >= len(h.flatItems) {
		return nil
	}
	item := h.flatItems[h.cursor]
	if item.Type == session.ItemTypeSession {
		return item.Session
	}
	return nil
}

// statusWorker runs in a background goroutine (Priority 1C)
// It receives status update requests and processes them without blocking the UI
func (h *Home) statusWorker() {
	defer close(h.statusWorkerDone)

	for {
		select {
		case <-h.ctx.Done():
			return
		case req := <-h.statusTrigger:
			// Panic recovery to prevent worker death from killing status updates
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("STATUS WORKER PANIC (recovered): %v", r)
					}
				}()
				h.processStatusUpdate(req)
			}()
		}
	}
}

// triggerStatusUpdate sends a non-blocking request to the background worker
// If the worker is busy, the request is dropped (next tick will retry)
func (h *Home) triggerStatusUpdate() {
	// Build list of session IDs from flatItems for visible detection
	flatItemIDs := make([]string, 0, len(h.flatItems))
	for _, item := range h.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil {
			flatItemIDs = append(flatItemIDs, item.Session.ID)
		}
	}

	visibleHeight := h.height - 8
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	req := statusUpdateRequest{
		viewOffset:    h.viewOffset,
		visibleHeight: visibleHeight,
		flatItemIDs:   flatItemIDs,
	}

	// Non-blocking send - if worker is busy, skip this tick
	select {
	case h.statusTrigger <- req:
		// Request sent successfully
	default:
		// Worker busy, will retry next tick
	}
}

// processStatusUpdate implements round-robin status updates (Priority 1A + 1B)
// Called by the background worker goroutine
// Instead of updating ALL sessions every tick (which causes lag with 100+ sessions),
// we update in batches:
//   - Always update visible sessions first (ensures UI responsiveness)
//   - Round-robin through remaining sessions (spreads CPU load over time)
//
// Performance: With 100 sessions, updating all takes ~5-10s of cumulative time per tick.
// With batching, we update ~10-15 sessions per tick, keeping each tick under 100ms.
func (h *Home) processStatusUpdate(req statusUpdateRequest) {
	const batchSize = 5 // Non-visible sessions to update per tick

	// Take a snapshot of instances under read lock (thread-safe)
	h.instancesMu.RLock()
	if len(h.instances) == 0 {
		h.instancesMu.RUnlock()
		return
	}
	instancesCopy := make([]*session.Instance, len(h.instances))
	copy(instancesCopy, h.instances)
	h.instancesMu.RUnlock()

	// Build set of visible session IDs for quick lookup
	visibleIDs := make(map[string]bool)

	// Find visible sessions based on viewOffset and flatItemIDs
	for i := req.viewOffset; i < len(req.flatItemIDs) && i < req.viewOffset+req.visibleHeight; i++ {
		visibleIDs[req.flatItemIDs[i]] = true
	}

	// Track which sessions we've updated this tick
	updated := make(map[string]bool)

	// Step 1: Always update visible sessions (Priority 1B - visible first)
	for _, inst := range instancesCopy {
		if visibleIDs[inst.ID] {
			// UpdateStatus is thread-safe (uses internal mutex)
			_ = inst.UpdateStatus() // Ignore errors in background worker
			updated[inst.ID] = true
		}
	}

	// Step 2: Round-robin through non-visible sessions (Priority 1A - batching)
	remaining := batchSize
	startIdx := int(h.statusUpdateIndex.Load())
	instanceCount := len(instancesCopy)

	for i := 0; i < instanceCount && remaining > 0; i++ {
		idx := (startIdx + i) % instanceCount
		inst := instancesCopy[idx]

		// Skip if already updated (visible)
		if updated[inst.ID] {
			continue
		}

		_ = inst.UpdateStatus() // Ignore errors in background worker
		remaining--
		h.statusUpdateIndex.Store(int32((idx + 1) % instanceCount))
	}
}

// Update handles messages
func (h *Home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
		h.updateSizes()
		h.syncViewport() // Recalculate viewport when window size changes
		return h, nil

	case loadSessionsMsg:
		if msg.err != nil {
			h.err = msg.err
		} else {
			h.instancesMu.Lock()
			h.instances = msg.instances
			h.instancesMu.Unlock()
			// Preserve existing group tree structure if it exists
			// Only create new tree on initial load (when groupTree has no groups)
			if h.groupTree.GroupCount() == 0 {
				// Initial load - use stored groups if available
				if len(msg.groups) > 0 {
					h.groupTree = session.NewGroupTreeWithGroups(h.instances, msg.groups)
				} else {
					h.groupTree = session.NewGroupTree(h.instances)
				}
			} else {
				// Refresh - update existing tree with loaded sessions
				h.groupTree.SyncWithInstances(h.instances)
			}
			h.rebuildFlatItems()
			h.search.SetItems(h.instances)
			// Trigger immediate preview fetch for initial selection (mutex-protected)
			if selected := h.getSelectedSession(); selected != nil {
				h.previewCacheMu.Lock()
				h.previewFetchingID = selected.ID
				h.previewCacheMu.Unlock()
				return h, h.fetchPreview(selected)
			}
		}
		return h, nil

	case sessionCreatedMsg:
		if msg.err != nil {
			h.err = msg.err
		} else {
			h.instancesMu.Lock()
			h.instances = append(h.instances, msg.instance)
			h.instancesMu.Unlock()
			// Add to existing group tree instead of rebuilding
			h.groupTree.AddSession(msg.instance)
			h.rebuildFlatItems()
			h.search.SetItems(h.instances)
			// Save both instances AND groups (critical fix: was losing groups!)
			h.saveInstances()
		}
		return h, nil

	case sessionForkedMsg:
		if msg.err != nil {
			h.err = msg.err
		} else {
			h.instancesMu.Lock()
			h.instances = append(h.instances, msg.instance)
			h.instancesMu.Unlock()
			// Add to existing group tree instead of rebuilding
			h.groupTree.AddSession(msg.instance)
			h.rebuildFlatItems()
			h.search.SetItems(h.instances)
			// Save both instances AND groups
			h.saveInstances()
		}
		return h, nil

	case sessionDeletedMsg:
		// Report kill error if any (session may still be running in tmux)
		if msg.killErr != nil {
			h.err = fmt.Errorf("warning: tmux session may still be running: %w", msg.killErr)
		}

		// Find and remove from list
		var deletedInstance *session.Instance
		h.instancesMu.Lock()
		for i, s := range h.instances {
			if s.ID == msg.deletedID {
				deletedInstance = s
				h.instances = append(h.instances[:i], h.instances[i+1:]...)
				break
			}
		}
		h.instancesMu.Unlock()
		// Remove from group tree (preserves empty groups)
		if deletedInstance != nil {
			h.groupTree.RemoveSession(deletedInstance)
		}
		h.rebuildFlatItems()
		// Update search items
		h.search.SetItems(h.instances)
		// Save both instances AND groups (critical fix: was losing groups!)
		h.saveInstances()
		return h, nil

	case sessionRestartedMsg:
		if msg.err != nil {
			h.err = fmt.Errorf("failed to restart session: %w", msg.err)
		} else {
			// Save the updated session state (new tmux session name)
			h.saveInstances()
		}
		return h, nil

	case updateCheckMsg:
		h.updateInfo = msg.info
		return h, nil

	case refreshMsg:
		return h, h.loadSessions

	case statusUpdateMsg:
		// Clear attach flag - we've returned from the attached session
		h.isAttaching = false

		// Immediate status update without reloading from storage
		// Used when returning from attached session
		for _, inst := range h.instances {
			if err := inst.UpdateStatus(); err != nil {
				// Log error but don't fail - other sessions still need updating
				h.err = fmt.Errorf("status update failed for %s: %w", inst.Title, err)
			}
		}
		// Save state after returning from attached session to persist acknowledged state
		h.saveInstances()
		return h, nil

	case previewFetchedMsg:
		// Async preview content received - update cache
		// Protect both previewFetchingID and previewCache with the same mutex
		h.previewCacheMu.Lock()
		h.previewFetchingID = ""
		if msg.err == nil {
			h.previewCache[msg.sessionID] = msg.content
		}
		h.previewCacheMu.Unlock()
		return h, nil

	case tickMsg:
		// Background status updates (Priority 1C optimization)
		// Triggers background worker to update session statuses without blocking UI
		// Worker implements round-robin batching (Priority 1A + 1B)
		h.triggerStatusUpdate()

		// Fetch preview for currently selected session (if not already fetching)
		// Protect previewFetchingID access with mutex
		var previewCmd tea.Cmd
		if selected := h.getSelectedSession(); selected != nil {
			h.previewCacheMu.Lock()
			if h.previewFetchingID != selected.ID {
				h.previewFetchingID = selected.ID
				previewCmd = h.fetchPreview(selected)
			}
			h.previewCacheMu.Unlock()
		}
		return h, tea.Batch(h.tick(), previewCmd)

	case tea.KeyMsg:
		// Handle overlays first
		// Help overlay takes priority (any key closes it)
		if h.helpOverlay.IsVisible() {
			h.helpOverlay, _ = h.helpOverlay.Update(msg)
			return h, nil
		}
		if h.search.IsVisible() {
			return h.handleSearchKey(msg)
		}
		if h.newDialog.IsVisible() {
			return h.handleNewDialogKey(msg)
		}
		if h.groupDialog.IsVisible() {
			return h.handleGroupDialogKey(msg)
		}
		if h.forkDialog.IsVisible() {
			return h.handleForkDialogKey(msg)
		}
		if h.confirmDialog.IsVisible() {
			return h.handleConfirmDialogKey(msg)
		}

		// Main view keys
		return h.handleMainKey(msg)
	}

	return h, tea.Batch(cmds...)
}

// handleSearchKey handles keys when search is visible
func (h *Home) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		selected := h.search.Selected()
		if selected != nil {
			// Ensure the session's group AND all parent groups are expanded so it's visible
			if selected.GroupPath != "" {
				h.groupTree.ExpandGroupWithParents(selected.GroupPath)
			}
			h.rebuildFlatItems()

			// Find the session in flatItems (not instances) and set cursor
			for i, item := range h.flatItems {
				if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == selected.ID {
					h.cursor = i
					h.syncViewport() // Ensure the cursor is visible in the viewport
					break
				}
			}
		}
		h.search.Hide()
		return h, nil
	case "esc":
		h.search.Hide()
		return h, nil
	}

	var cmd tea.Cmd
	h.search, cmd = h.search.Update(msg)
	return h, cmd
}

// handleNewDialogKey handles keys when new dialog is visible
func (h *Home) handleNewDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate before creating session
		if validationErr := h.newDialog.Validate(); validationErr != "" {
			h.err = fmt.Errorf("validation error: %s", validationErr)
			return h, nil
		}

		// Create session (enter works from any field)
		name, path, command := h.newDialog.GetValues()
		groupPath := h.newDialog.GetSelectedGroup()
		h.newDialog.Hide()
		h.err = nil // Clear any previous validation error
		return h, h.createSessionInGroup(name, path, command, groupPath)

	case "esc":
		h.newDialog.Hide()
		h.err = nil // Clear any validation error
		return h, nil
	}

	var cmd tea.Cmd
	h.newDialog, cmd = h.newDialog.Update(msg)
	return h, cmd
}

// handleMainKey handles keys in main view
func (h *Home) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		h.cancel() // Signal background worker to stop
		// Wait for background worker to finish (prevents race on shutdown)
		<-h.statusWorkerDone
		if h.logWatcher != nil {
			h.logWatcher.Close()
		}
		// Save both instances AND groups on quit (critical fix: was losing groups!)
		h.saveInstances()
		return h, tea.Quit

	case "up", "k":
		if h.cursor > 0 {
			h.cursor--
			h.syncViewport()
			// Trigger immediate preview fetch for new selection (mutex-protected)
			if selected := h.getSelectedSession(); selected != nil {
				h.previewCacheMu.Lock()
				needsFetch := h.previewFetchingID != selected.ID
				if needsFetch {
					h.previewFetchingID = selected.ID
				}
				h.previewCacheMu.Unlock()
				if needsFetch {
					return h, h.fetchPreview(selected)
				}
			}
		}
		return h, nil

	case "down", "j":
		if h.cursor < len(h.flatItems)-1 {
			h.cursor++
			h.syncViewport()
			// Trigger immediate preview fetch for new selection (mutex-protected)
			if selected := h.getSelectedSession(); selected != nil {
				h.previewCacheMu.Lock()
				needsFetch := h.previewFetchingID != selected.ID
				if needsFetch {
					h.previewFetchingID = selected.ID
				}
				h.previewCacheMu.Unlock()
				if needsFetch {
					return h, h.fetchPreview(selected)
				}
			}
		}
		return h, nil

	case "enter":
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				if item.Session.Exists() {
					h.isAttaching = true // Prevent View() output during transition
					return h, h.attachSession(item.Session)
				}
			} else if item.Type == session.ItemTypeGroup {
				// Toggle group on enter
				h.groupTree.ToggleGroup(item.Path)
				h.rebuildFlatItems()
			}
		}
		return h, nil

	case "tab", "l", "right":
		// Expand/collapse group or expand if on session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.ToggleGroup(item.Path)
				h.rebuildFlatItems()
			}
		}
		return h, nil

	case "h", "left":
		// Collapse group
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.CollapseGroup(item.Path)
				h.rebuildFlatItems()
			} else if item.Type == session.ItemTypeSession {
				// Move cursor to parent group
				h.groupTree.CollapseGroup(item.Path)
				h.rebuildFlatItems()
				// Find the group in flatItems
				for i, fi := range h.flatItems {
					if fi.Type == session.ItemTypeGroup && fi.Path == item.Path {
						h.cursor = i
						break
					}
				}
			}
		}
		return h, nil

	case "shift+up", "K":
		// Move item up
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.MoveGroupUp(item.Path)
			} else if item.Type == session.ItemTypeSession {
				h.groupTree.MoveSessionUp(item.Session)
			}
			h.rebuildFlatItems()
			if h.cursor > 0 {
				h.cursor--
			}
			h.saveInstances()
		}
		return h, nil

	case "shift+down", "J":
		// Move item down
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.MoveGroupDown(item.Path)
			} else if item.Type == session.ItemTypeSession {
				h.groupTree.MoveSessionDown(item.Session)
			}
			h.rebuildFlatItems()
			if h.cursor < len(h.flatItems)-1 {
				h.cursor++
			}
			h.saveInstances()
		}
		return h, nil

	case "m":
		// Move session to different group
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession {
				h.groupDialog.ShowMove(h.groupTree.GetGroupNames())
			}
		}
		return h, nil

	case "f":
		// Quick fork session (same title with " (fork)" suffix)
		// Only available when session has a valid Claude session ID
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.CanFork() {
				return h, h.quickForkSession(item.Session)
			}
		}
		return h, nil

	case "F", "shift+f":
		// Fork with dialog (customize title and group)
		// Only available when session has a valid Claude session ID
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.CanFork() {
				return h, h.forkSessionWithDialog(item.Session)
			}
		}
		return h, nil

	case "g":
		// Create new group (or subgroup if a group is selected)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				// Create subgroup under selected group
				h.groupDialog.ShowCreateSubgroup(item.Group.Path, item.Group.Name)
				return h, nil
			}
		}
		// Create root-level group
		h.groupDialog.Show()
		return h, nil

	case "R", "shift+r":
		// Rename group or session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupDialog.ShowRename(item.Path, item.Group.Name)
			} else if item.Type == session.ItemTypeSession && item.Session != nil {
				h.groupDialog.ShowRenameSession(item.Session.ID, item.Session.Title)
			}
		}
		return h, nil

	case "/":
		h.search.Show()
		return h, nil

	case "?":
		h.helpOverlay.SetSize(h.width, h.height)
		h.helpOverlay.Show()
		return h, nil

	case "n":
		// Collect unique project paths from existing sessions
		pathSet := make(map[string]bool)
		var paths []string
		for _, inst := range h.instances {
			if inst.ProjectPath != "" && !pathSet[inst.ProjectPath] {
				pathSet[inst.ProjectPath] = true
				paths = append(paths, inst.ProjectPath)
			}
		}
		h.newDialog.SetPathSuggestions(paths)

		// Apply user's preferred default tool from config
		h.newDialog.SetDefaultTool(session.GetDefaultTool())

		// Auto-select parent group from current cursor position
		groupPath := session.DefaultGroupName
		groupName := session.DefaultGroupName
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				groupPath = item.Group.Path
				groupName = item.Group.Name
			} else if item.Type == session.ItemTypeSession {
				// Use the session's group
				groupPath = item.Path
				if group, exists := h.groupTree.Groups[groupPath]; exists {
					groupName = group.Name
				}
			}
		}
		h.newDialog.ShowInGroup(groupPath, groupName)
		return h, nil

	case "d":
		// Show confirmation dialog before deletion (prevents accidental deletion)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				h.confirmDialog.ShowDeleteSession(item.Session.ID, item.Session.Title)
			} else if item.Type == session.ItemTypeGroup && item.Path != session.DefaultGroupName {
				h.confirmDialog.ShowDeleteGroup(item.Path, item.Group.Name)
			}
		}
		return h, nil

	case "i":
		return h, h.importSessions

	case "r":
		return h, h.loadSessions

	case "u":
		// Mark session as unread (change idle → waiting)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				tmuxSess := item.Session.GetTmuxSession()
				if tmuxSess != nil {
					tmuxSess.ResetAcknowledged()
					_ = item.Session.UpdateStatus()
					h.saveInstances()
				}
			}
		}
		return h, nil

	case "S":
		// Restart/Start a dead/errored session (recreate tmux session)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				if item.Session.CanRestart() {
					return h, h.restartSession(item.Session)
				}
			}
		}
		return h, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Quick jump to Nth root group (1-indexed)
		targetNum := int(msg.String()[0] - '0') // Convert "1" -> 1, "2" -> 2, etc.
		h.jumpToRootGroup(targetNum)
		return h, nil
	}

	return h, nil
}

// handleConfirmDialogKey handles keys when confirmation dialog is visible
func (h *Home) handleConfirmDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// User confirmed - perform the deletion
		switch h.confirmDialog.GetConfirmType() {
		case ConfirmDeleteSession:
			sessionID := h.confirmDialog.GetTargetID()
			for _, inst := range h.instances {
				if inst.ID == sessionID {
					h.confirmDialog.Hide()
					return h, h.deleteSession(inst)
				}
			}
		case ConfirmDeleteGroup:
			groupPath := h.confirmDialog.GetTargetID()
			h.groupTree.DeleteGroup(groupPath)
			h.instancesMu.Lock()
			h.instances = h.groupTree.GetAllInstances()
			h.instancesMu.Unlock()
			h.rebuildFlatItems()
			h.saveInstances()
		}
		h.confirmDialog.Hide()
		return h, nil

	case "n", "N", "esc":
		// User cancelled
		h.confirmDialog.Hide()
		return h, nil
	}

	return h, nil
}

// handleGroupDialogKey handles keys when group dialog is visible
func (h *Home) handleGroupDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate before proceeding
		if validationErr := h.groupDialog.Validate(); validationErr != "" {
			h.err = fmt.Errorf("validation error: %s", validationErr)
			return h, nil
		}
		h.err = nil // Clear any previous validation error

		switch h.groupDialog.Mode() {
		case GroupDialogCreate:
			name := h.groupDialog.GetValue()
			if name != "" {
				if h.groupDialog.HasParent() {
					// Create subgroup under parent
					parentPath := h.groupDialog.GetParentPath()
					h.groupTree.CreateSubgroup(parentPath, name)
				} else {
					// Create root-level group
					h.groupTree.CreateGroup(name)
				}
				h.rebuildFlatItems()
				h.saveInstances() // Persist the new group
			}
		case GroupDialogRename:
			name := h.groupDialog.GetValue()
			if name != "" {
				h.groupTree.RenameGroup(h.groupDialog.GetGroupPath(), name)
				h.instancesMu.Lock()
				h.instances = h.groupTree.GetAllInstances()
				h.instancesMu.Unlock()
				h.rebuildFlatItems()
				h.saveInstances()
			}
		case GroupDialogMove:
			groupName := h.groupDialog.GetSelectedGroup()
			if groupName != "" && h.cursor < len(h.flatItems) {
				item := h.flatItems[h.cursor]
				if item.Type == session.ItemTypeSession {
					// Find the group path from name
					for _, g := range h.groupTree.GroupList {
						if g.Name == groupName {
							h.groupTree.MoveSessionToGroup(item.Session, g.Path)
							h.instancesMu.Lock()
							h.instances = h.groupTree.GetAllInstances()
							h.instancesMu.Unlock()
							h.rebuildFlatItems()
							h.saveInstances()
							break
						}
					}
				}
			}
		case GroupDialogRenameSession:
			newName := h.groupDialog.GetValue()
			if newName != "" {
				sessionID := h.groupDialog.GetSessionID()
				// Find and rename the session
				for _, inst := range h.instances {
					if inst.ID == sessionID {
						inst.Title = newName
						break
					}
				}
				h.rebuildFlatItems()
				h.saveInstances()
			}
		}
		h.groupDialog.Hide()
		return h, nil
	case "esc":
		h.groupDialog.Hide()
		h.err = nil // Clear any validation error
		return h, nil
	}

	var cmd tea.Cmd
	h.groupDialog, cmd = h.groupDialog.Update(msg)
	return h, cmd
}

// handleForkDialogKey handles keyboard input for the fork dialog
func (h *Home) handleForkDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Get fork parameters from dialog
		title, groupPath := h.forkDialog.GetValues()
		if title == "" {
			h.err = fmt.Errorf("session name cannot be empty")
			return h, nil
		}
		h.err = nil // Clear any previous error

		// Find the currently selected session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				h.forkDialog.Hide()
				return h, h.forkSessionCmd(item.Session, title, groupPath)
			}
		}
		h.forkDialog.Hide()
		return h, nil

	case "esc":
		h.forkDialog.Hide()
		h.err = nil // Clear any error
		return h, nil
	}

	var cmd tea.Cmd
	h.forkDialog, cmd = h.forkDialog.Update(msg)
	return h, cmd
}

// saveInstances saves instances to storage
func (h *Home) saveInstances() {
	if h.storage != nil {
		// Save both instances and groups (including empty ones)
		if err := h.storage.SaveWithGroups(h.instances, h.groupTree); err != nil {
			h.err = fmt.Errorf("failed to save: %w", err)
		}
	}
}

// createSessionInGroup creates a new session in a specific group
func (h *Home) createSessionInGroup(name, path, command, groupPath string) tea.Cmd {
	return func() tea.Msg {
		// Check tmux availability before creating session
		if err := tmux.IsTmuxAvailable(); err != nil {
			return sessionCreatedMsg{err: fmt.Errorf("cannot create session: %w", err)}
		}

		// Determine tool from command for proper session initialization
		// When tool is "claude", session ID will be detected from files after start
		tool := "shell"
		switch command {
		case "claude":
			tool = "claude"
		case "gemini":
			tool = "gemini"
		case "aider":
			tool = "aider"
		case "codex":
			tool = "codex"
		}

		var inst *session.Instance
		if groupPath != "" {
			inst = session.NewInstanceWithGroupAndTool(name, path, groupPath, tool)
		} else {
			inst = session.NewInstanceWithTool(name, path, tool)
		}
		inst.Command = command
		if err := inst.Start(); err != nil {
			return sessionCreatedMsg{err: err}
		}
		return sessionCreatedMsg{instance: inst}
	}
}

// quickForkSession performs a quick fork with default title suffix " (fork)"
func (h *Home) quickForkSession(source *session.Instance) tea.Cmd {
	if source == nil {
		return nil
	}
	// Use source title with " (fork)" suffix
	title := source.Title + " (fork)"
	groupPath := source.GroupPath
	return h.forkSessionCmd(source, title, groupPath)
}

// forkSessionWithDialog opens the fork dialog to customize title and group
func (h *Home) forkSessionWithDialog(source *session.Instance) tea.Cmd {
	if source == nil {
		return nil
	}
	// Pre-populate dialog with source session info
	h.forkDialog.Show(source.Title, source.ProjectPath, source.GroupPath)
	return nil
}

// forkSessionCmd creates a forked session with the given title and group
func (h *Home) forkSessionCmd(source *session.Instance, title, groupPath string) tea.Cmd {
	if source == nil {
		return nil
	}
	return func() tea.Msg {
		// Check tmux availability before forking
		if err := tmux.IsTmuxAvailable(); err != nil {
			return sessionForkedMsg{err: fmt.Errorf("cannot fork session: %w", err)}
		}

		// Use CreateForkedInstance to get the proper fork command
		inst, _, err := source.CreateForkedInstance(title, groupPath)
		if err != nil {
			return sessionForkedMsg{err: fmt.Errorf("cannot create forked instance: %w", err)}
		}

		// Start the forked session
		if err := inst.Start(); err != nil {
			return sessionForkedMsg{err: err}
		}

		// Wait for Claude to create the new session file (fork creates new UUID)
		// Give Claude up to 3 seconds to initialize and write the session file
		if inst.Tool == "claude" {
			_ = inst.WaitForClaudeSession(3 * time.Second)
		}

		return sessionForkedMsg{instance: inst}
	}
}

// sessionDeletedMsg signals that a session was deleted
type sessionDeletedMsg struct {
	deletedID string
	killErr   error // Error from Kill() if any
}

// deleteSession deletes a session
func (h *Home) deleteSession(inst *session.Instance) tea.Cmd {
	id := inst.ID
	return func() tea.Msg {
		killErr := inst.Kill()
		return sessionDeletedMsg{deletedID: id, killErr: killErr}
	}
}

// sessionRestartedMsg signals that a session was restarted
type sessionRestartedMsg struct {
	sessionID string
	err       error
}

// restartSession restarts a dead/errored session by creating a new tmux session
func (h *Home) restartSession(inst *session.Instance) tea.Cmd {
	id := inst.ID
	return func() tea.Msg {
		err := inst.Restart()
		return sessionRestartedMsg{sessionID: id, err: err}
	}
}

// attachSession attaches to a session using custom PTY with Ctrl+Q detection
func (h *Home) attachSession(inst *session.Instance) tea.Cmd {
	tmuxSess := inst.GetTmuxSession()
	if tmuxSess == nil {
		return nil
	}

	// NOTE: We DON'T call Acknowledge() here. Setting acknowledged=true before attach
	// would cause brief "idle" status if a poll happens before content changes.
	// The proper acknowledgment happens in AcknowledgeWithSnapshot() AFTER detach,
	// which baselines the content hash the user saw.

	// Use tea.Exec with a custom command that runs our Attach method
	// On return, immediately update all session statuses (don't reload from storage
	// which would lose the tmux session state)
	return tea.Exec(attachCmd{session: tmuxSess}, func(err error) tea.Msg {
		// Clear screen with synchronized output for atomic rendering
		fmt.Print(syncOutputBegin + clearScreen + syncOutputEnd)

		// Baseline the content the user just saw to avoid a green flash on return
		tmuxSess.AcknowledgeWithSnapshot()
		return statusUpdateMsg{}
	})
}

// attachCmd implements tea.ExecCommand for custom PTY attach
type attachCmd struct {
	session *tmux.Session
}

func (a attachCmd) Run() error {
	// Clear screen with synchronized output for atomic rendering (prevents flicker)
	// Begin sync mode → clear screen → end sync mode ensures single-frame update
	fmt.Print(syncOutputBegin + clearScreen + syncOutputEnd)

	ctx := context.Background()
	return a.session.Attach(ctx)
}

func (a attachCmd) SetStdin(r io.Reader)  {}
func (a attachCmd) SetStdout(w io.Writer) {}
func (a attachCmd) SetStderr(w io.Writer) {}

// importSessions imports existing tmux sessions
func (h *Home) importSessions() tea.Msg {
	discovered, err := session.DiscoverExistingTmuxSessions(h.instances)
	if err != nil {
		return loadSessionsMsg{err: err}
	}

	h.instancesMu.Lock()
	h.instances = append(h.instances, discovered...)
	instancesCopy := make([]*session.Instance, len(h.instances))
	copy(instancesCopy, h.instances)
	h.instancesMu.Unlock()

	// Add discovered sessions to group tree before saving
	for _, inst := range discovered {
		h.groupTree.AddSession(inst)
	}
	// Save both instances AND groups (critical fix: was losing groups!)
	h.saveInstances()
	return loadSessionsMsg{instances: instancesCopy}
}

// countSessionStatuses counts sessions by status for the logo display
func (h *Home) countSessionStatuses() (running, waiting, idle int) {
	for _, inst := range h.instances {
		switch inst.Status {
		case session.StatusRunning:
			running++
		case session.StatusWaiting:
			waiting++
		case session.StatusIdle:
			idle++
			// StatusError is counted as neither - will show as idle in logo
		}
	}
	return running, waiting, idle
}

// updateSizes updates component sizes
func (h *Home) updateSizes() {
	h.search.SetSize(h.width, h.height)
	h.newDialog.SetSize(h.width, h.height)
	h.groupDialog.SetSize(h.width, h.height)
	h.confirmDialog.SetSize(h.width, h.height)
}

// View renders the UI
func (h *Home) View() string {
	// CRITICAL: Return empty during attach to prevent View() output leakage
	// (Bubble Tea Issue #431 - View gets printed to stdout during tea.Exec)
	if h.isAttaching {
		return ""
	}

	if h.width == 0 {
		return "Loading..."
	}

	// Overlays take full screen
	if h.helpOverlay.IsVisible() {
		return h.helpOverlay.View()
	}
	if h.search.IsVisible() {
		return h.search.View()
	}
	if h.newDialog.IsVisible() {
		return h.newDialog.View()
	}
	if h.groupDialog.IsVisible() {
		return h.groupDialog.View()
	}
	if h.forkDialog.IsVisible() {
		return h.forkDialog.View()
	}
	if h.confirmDialog.IsVisible() {
		return h.confirmDialog.View()
	}

	var b strings.Builder

	// ═══════════════════════════════════════════════════════════════════
	// HEADER BAR
	// ═══════════════════════════════════════════════════════════════════
	// Calculate real session status counts for logo
	running, waiting, idle := h.countSessionStatuses()
	logo := RenderLogoCompact(running, waiting, idle)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Background(ColorSurface).
		Padding(0, 1)

	// Show profile in title if not default
	titleText := "Agent Deck"
	if h.profile != "" && h.profile != session.DefaultProfile {
		profileStyle := lipgloss.NewStyle().
			Foreground(ColorCyan).
			Background(ColorSurface)
		titleText = "Agent Deck " + profileStyle.Render("["+h.profile+"]")
	}
	title := titleStyle.Render(titleText)

	// Stats
	stats := lipgloss.NewStyle().Foreground(ColorTextDim).Render(
		fmt.Sprintf(" %d groups • %d sessions", h.groupTree.GroupCount(), h.groupTree.SessionCount()))

	// Version badge (right-aligned, subtle)
	versionStyle := lipgloss.NewStyle().
		Foreground(ColorComment). // Dim gray, doesn't compete with main content
		Italic(true)
	versionBadge := versionStyle.Render("v" + Version)

	// Fill remaining header space, placing version at the right
	headerLeft := lipgloss.JoinHorizontal(lipgloss.Left, logo, " ", title, stats)
	// Calculate padding between stats and version
	headerPadding := h.width - lipgloss.Width(headerLeft) - lipgloss.Width(versionBadge) - 1
	if headerPadding < 1 {
		headerPadding = 1
	}
	headerContent := headerLeft + strings.Repeat(" ", headerPadding) + versionBadge

	headerBar := lipgloss.NewStyle().
		Background(ColorSurface).
		Width(h.width).
		Render(headerContent)

	b.WriteString(headerBar)
	b.WriteString("\n")

	// ═══════════════════════════════════════════════════════════════════
	// UPDATE BANNER (if update available)
	// ═══════════════════════════════════════════════════════════════════
	updateBannerHeight := 0
	if h.updateInfo != nil && h.updateInfo.Available {
		updateBannerHeight = 1
		updateStyle := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorYellow).
			Bold(true).
			Width(h.width).
			Align(lipgloss.Center)
		updateText := fmt.Sprintf(" ⬆ Update available: v%s → v%s (run: agent-deck update) ",
			h.updateInfo.CurrentVersion, h.updateInfo.LatestVersion)
		b.WriteString(updateStyle.Render(updateText))
		b.WriteString("\n")
	}

	// ═══════════════════════════════════════════════════════════════════
	// MAIN CONTENT AREA
	// ═══════════════════════════════════════════════════════════════════
	helpBarHeight := 3                                              // Help bar takes 3 lines
	contentHeight := h.height - 2 - helpBarHeight - updateBannerHeight // -2 for header, -helpBarHeight for help

	// Calculate panel widths (35% left, 65% right for more preview space)
	leftWidth := int(float64(h.width) * 0.35)
	rightWidth := h.width - leftWidth - 3 // -3 for separator

	// Build left panel (session list) with title
	leftTitle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true).
		Render("SESSIONS")
	leftContent := h.renderSessionList(contentHeight - 2) // -2 for title
	leftPanel := lipgloss.JoinVertical(lipgloss.Left, leftTitle, leftContent)
	leftPanel = lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Render(leftPanel)

	// Build right panel (preview) with title
	rightTitle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true).
		Render("PREVIEW")
	rightContent := h.renderPreviewPane(rightWidth, contentHeight-2) // -2 for title
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, rightTitle, rightContent)
	rightPanel = lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Render(rightPanel)

	// Build separator
	separatorStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	separatorLines := make([]string, contentHeight)
	for i := range separatorLines {
		separatorLines[i] = separatorStyle.Render(" │ ")
	}
	separator := strings.Join(separatorLines, "\n")

	// Join panels horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)
	b.WriteString(mainContent)
	b.WriteString("\n")

	// ═══════════════════════════════════════════════════════════════════
	// HELP BAR (context-aware shortcuts)
	// ═══════════════════════════════════════════════════════════════════
	helpBar := h.renderHelpBar()
	b.WriteString(helpBar)

	// Error display
	if h.err != nil {
		errMsg := ErrorStyle.Render("⚠ " + h.err.Error())
		b.WriteString("\n")
		b.WriteString(errMsg)
	}

	// Storage warning (persistent until resolved)
	if h.storageWarning != "" {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(h.storageWarning))
	}

	return b.String()
}

// renderHelpBar renders context-aware keyboard shortcuts
func (h *Home) renderHelpBar() string {
	// Determine context
	var contextHints []string
	var contextTitle string

	if len(h.flatItems) == 0 {
		contextTitle = "No sessions"
		contextHints = []string{
			h.helpKey("n", "New session"),
			h.helpKey("i", "Import tmux"),
			h.helpKey("g", "New group"),
			h.helpKey("q", "Quit"),
		}
	} else if h.cursor < len(h.flatItems) {
		item := h.flatItems[h.cursor]
		if item.Type == session.ItemTypeGroup {
			contextTitle = "Group selected"
			contextHints = []string{
				h.helpKey("Tab/l", "Toggle"),
				h.helpKey("R", "Rename"),
				h.helpKey("d", "Delete"),
				h.helpKey("g", "New subgroup"),
				h.helpKey("n", "New session"),
			}
		} else {
			contextTitle = "Session selected"
			contextHints = []string{
				h.helpKey("Enter", "Attach"),
			}
			// Only show fork hints if session has a valid Claude session ID
			if item.Session != nil && item.Session.CanFork() {
				contextHints = append(contextHints,
					h.helpKey("f", "Fork"),
					h.helpKey("F", "Fork (custom)"),
				)
			}
			contextHints = append(contextHints,
				h.helpKey("R", "Rename"),
				h.helpKey("m", "Move to group"),
				h.helpKey("d", "Delete"),
			)
		}
	}

	// Build help bar
	border := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", h.width))

	// Context title
	ctxStyle := lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)

	// Build shortcuts line
	shortcutsLine := strings.Join(contextHints, "  ")

	// Global shortcuts (always shown)
	globalHints := lipgloss.NewStyle().Foreground(ColorTextDim).Render(
		"  │  ↑↓/jk Navigate  /Search  Ctrl+Q Detach  q Quit")

	helpContent := lipgloss.JoinHorizontal(lipgloss.Left,
		ctxStyle.Render(contextTitle+": "),
		shortcutsLine,
		globalHints,
	)

	return lipgloss.JoinVertical(lipgloss.Left, border, helpContent)
}

// helpKey formats a keyboard shortcut for the help bar
func (h *Home) helpKey(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)
	return keyStyle.Render(key) + descStyle.Render(" "+desc)
}

// renderSessionList renders the left panel with hierarchical session list
func (h *Home) renderSessionList(height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 {
		// Large logo for empty state - shows real status (all idle when no sessions)
		running, waiting, idle := h.countSessionStatuses()
		largeLogo := RenderLogoLarge(running, waiting, idle)

		// App title with profile
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)
		appTitle := titleStyle.Render("Agent Deck")
		if h.profile != "" && h.profile != session.DefaultProfile {
			profileStyle := lipgloss.NewStyle().Foreground(ColorCyan)
			appTitle += " " + profileStyle.Render("["+h.profile+"]")
		}

		// Subtitle
		subtitleStyle := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true)
		subtitle := subtitleStyle.Render("Terminal Session Manager")

		// Instructions
		instructions := lipgloss.NewStyle().Foreground(ColorAccent).Render("n") + " Create new\n" +
			lipgloss.NewStyle().Foreground(ColorAccent).Render("i") + " Import from tmux\n" +
			lipgloss.NewStyle().Foreground(ColorAccent).Render("g") + " Create group"

		// Combine all elements
		content := lipgloss.JoinVertical(lipgloss.Center,
			largeLogo,
			"",
			appTitle,
			subtitle,
			"",
			instructions,
		)

		emptyBox := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 3).
			Render(content)
		return emptyBox
	}

	// Render items starting from viewOffset
	visibleCount := 0
	maxVisible := height - 1 // Leave room for scrolling indicator
	if maxVisible < 1 {
		maxVisible = 1
	}

	// Show "more above" indicator if scrolled down
	if h.viewOffset > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d above", h.viewOffset)))
		b.WriteString("\n")
		maxVisible-- // Account for the indicator line
	}

	for i := h.viewOffset; i < len(h.flatItems) && visibleCount < maxVisible; i++ {
		item := h.flatItems[i]
		h.renderItem(&b, item, i == h.cursor, i)
		visibleCount++
	}

	// Show "more below" indicator if there are more items
	remaining := len(h.flatItems) - (h.viewOffset + visibleCount)
	if remaining > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d below", remaining)))
	}

	return b.String()
}

// renderItem renders a single item (group or session) for the left panel
func (h *Home) renderItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	if item.Type == session.ItemTypeGroup {
		h.renderGroupItem(b, item, selected, itemIndex)
	} else {
		h.renderSessionItem(b, item, selected)
	}
}

// renderGroupItem renders a group header
func (h *Home) renderGroupItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	group := item.Group

	// Calculate indentation based on nesting level
	indent := strings.Repeat("  ", item.Level)
	if item.Level > 0 {
		indent = strings.Repeat("  ", item.Level-1) + "  ├─ "
	}

	// Expand/collapse indicator
	expandIcon := lipgloss.NewStyle().Foreground(ColorTextDim).Render("▼")
	if !group.Expanded {
		expandIcon = lipgloss.NewStyle().Foreground(ColorTextDim).Render("▶")
	}

	// Group name with count
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	countStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	if selected {
		nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorAccent)
		countStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent)
		expandIcon = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Render("▶")
		if group.Expanded {
			expandIcon = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorAccent).
				Render("▼")
		}
	}

	// Hotkey prefix for root groups (Level 0, positions 1-9)
	hotkeyPrefix := ""
	if item.Level == 0 {
		// Count how many root groups come before this one
		rootGroupNum := 0
		for i := 0; i <= itemIndex && i < len(h.flatItems); i++ {
			if h.flatItems[i].Type == session.ItemTypeGroup && h.flatItems[i].Level == 0 {
				rootGroupNum++
			}
		}
		// Only show hotkey for positions 1-9
		if rootGroupNum >= 1 && rootGroupNum <= 9 {
			hotkeyStyle := countStyle // Use same dim style as count
			hotkeyPrefix = hotkeyStyle.Render(fmt.Sprintf("[%d] ", rootGroupNum))
		}
	}

	sessionCount := len(group.Sessions)
	countStr := countStyle.Render(fmt.Sprintf(" (%d)", sessionCount))

	// Check if any session in group is running
	running := 0
	waiting := 0
	for _, sess := range group.Sessions {
		switch sess.Status {
		case session.StatusRunning:
			running++
		case session.StatusWaiting:
			waiting++
		}
	}

	// Status indicators
	statusStr := ""
	if running > 0 {
		statusStr += lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf(" ●%d", running))
	}
	if waiting > 0 {
		statusStr += lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf(" ◐%d", waiting))
	}

	// Build the row with proper indentation
	row := fmt.Sprintf("%s%s%s %s%s%s", indent, hotkeyPrefix, expandIcon, nameStyle.Render(group.Name), countStr, statusStr)
	b.WriteString(row)
	b.WriteString("\n")
}

// renderSessionItem renders a single session item for the left panel
func (h *Home) renderSessionItem(b *strings.Builder, item session.Item, selected bool) {
	inst := item.Session

	// Calculate indentation based on nesting level
	// Sessions are always under a group, so Level >= 1
	indent := strings.Repeat("  ", item.Level-1)
	treeLine := indent + "  ├─ "
	if selected {
		treeLine = indent + "  "
	}

	// Status indicator
	var statusIcon string
	var statusColor lipgloss.Color
	switch inst.Status {
	case session.StatusRunning:
		statusIcon = "●"
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusIcon = "◐"
		statusColor = ColorYellow
	case session.StatusIdle:
		statusIcon = "○"
		statusColor = ColorTextDim
	case session.StatusError:
		statusIcon = "✕"
		statusColor = ColorRed
	default:
		statusIcon = "?"
		statusColor = ColorTextDim
	}

	status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)

	// Title and tool
	titleStyle := lipgloss.NewStyle().Foreground(ColorText)
	toolStyle := lipgloss.NewStyle().Foreground(ColorPurple)

	if selected {
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorAccent)
		toolStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent)
		// Override tree line styling when selected
		treeLine = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Render(indent + "▶ ")
	}

	title := titleStyle.Render(inst.Title)
	tool := toolStyle.Render(fmt.Sprintf(" [%s]", inst.Tool))

	// Build row
	treeStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	if !selected {
		treeLine = treeStyle.Render(treeLine)
	}

	row := fmt.Sprintf("%s%s %s%s", treeLine, status, title, tool)
	b.WriteString(row)
	b.WriteString("\n")
}

// renderPreviewPane renders the right panel with live preview
func (h *Home) renderPreviewPane(width, height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 || h.cursor >= len(h.flatItems) {
		emptyBox := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2).
			Render("Select a session to see its terminal output")
		return emptyBox
	}

	item := h.flatItems[h.cursor]

	// If group is selected, show group info
	if item.Type == session.ItemTypeGroup {
		return h.renderGroupPreview(item.Group, width, height)
	}

	// Session preview
	selected := item.Session

	// Session info header box
	statusIcon := "○"
	statusColor := ColorTextDim
	switch selected.Status {
	case session.StatusRunning:
		statusIcon = "●"
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusIcon = "◐"
		statusColor = ColorYellow
	case session.StatusError:
		statusIcon = "✕"
		statusColor = ColorRed
	}

	// Header with session name and status
	statusBadge := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + string(selected.Status))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	b.WriteString(nameStyle.Render(selected.Title))
	b.WriteString("  ")
	b.WriteString(statusBadge)
	b.WriteString("\n")

	// Info line
	infoStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	pathStr := truncatePath(selected.ProjectPath, width-4)
	b.WriteString(infoStyle.Render("📁 " + pathStr))
	b.WriteString("\n")

	toolBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorPurple).
		Padding(0, 1).
		Render(selected.Tool)
	groupBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorCyan).
		Padding(0, 1).
		Render(selected.GroupPath)
	b.WriteString(toolBadge)
	b.WriteString(" ")
	b.WriteString(groupBadge)
	b.WriteString("\n\n")

	// Terminal output header
	termHeader := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("─── Terminal Output ───")
	b.WriteString(termHeader)
	b.WriteString("\n")

	// Terminal preview - use cached content (async fetching keeps View() pure)
	h.previewCacheMu.RLock()
	preview, hasCached := h.previewCache[selected.ID]
	h.previewCacheMu.RUnlock()
	if !hasCached {
		// Show loading indicator while waiting for async fetch
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true)
		b.WriteString(loadingStyle.Render("Loading preview..."))
	} else if preview == "" {
		emptyTerm := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true).
			Render("(terminal is empty)")
		b.WriteString(emptyTerm)
	} else {
		// Limit preview to available height
		lines := strings.Split(preview, "\n")
		maxLines := height - 8 // Account for header and info
		if maxLines < 1 {
			maxLines = 1
		}

		// Track if we're truncating from the top (for indicator)
		truncatedFromTop := len(lines) > maxLines
		truncatedCount := 0
		if truncatedFromTop {
			// Reserve one line for the truncation indicator
			maxLines--
			if maxLines < 1 {
				maxLines = 1
			}
			truncatedCount = len(lines) - maxLines
			lines = lines[len(lines)-maxLines:]
		}

		previewStyle := lipgloss.NewStyle().Foreground(ColorText)
		maxWidth := width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}

		// Show truncation indicator if content was cut from top
		if truncatedFromTop {
			truncIndicator := lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Italic(true).
				Render(fmt.Sprintf("⋮ %d more lines above", truncatedCount))
			b.WriteString(truncIndicator)
			b.WriteString("\n")
		}

		// Track consecutive empty lines to preserve some spacing
		consecutiveEmpty := 0
		const maxConsecutiveEmpty = 2 // Allow up to 2 consecutive empty lines

		for _, line := range lines {
			// Strip ANSI codes for accurate width measurement
			cleanLine := tmux.StripANSI(line)

			// Handle empty lines - preserve some for readability
			trimmed := strings.TrimSpace(cleanLine)
			if trimmed == "" {
				consecutiveEmpty++
				if consecutiveEmpty <= maxConsecutiveEmpty {
					b.WriteString("\n") // Preserve empty line
				}
				continue
			}
			consecutiveEmpty = 0 // Reset counter on non-empty line

			// Truncate based on display width (handles CJK, emoji correctly)
			displayWidth := runewidth.StringWidth(cleanLine)
			if displayWidth > maxWidth {
				cleanLine = runewidth.Truncate(cleanLine, maxWidth-3, "...")
			}

			b.WriteString(previewStyle.Render(cleanLine))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// truncatePath shortens a path to fit within maxLen characters
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen < 10 {
		maxLen = 10
	}
	// Show beginning and end: /Users/.../project
	return path[:maxLen/3] + "..." + path[len(path)-(maxLen*2/3-3):]
}

// renderGroupPreview renders the preview pane for a group
func (h *Home) renderGroupPreview(group *session.Group, width, height int) string {
	var b strings.Builder

	// Group name header
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	b.WriteString(nameStyle.Render("📁 " + group.Name))
	b.WriteString("\n\n")

	// Group stats
	running := 0
	waiting := 0
	idle := 0
	errored := 0
	for _, sess := range group.Sessions {
		switch sess.Status {
		case session.StatusRunning:
			running++
		case session.StatusWaiting:
			waiting++
		case session.StatusIdle:
			idle++
		case session.StatusError:
			errored++
		}
	}

	// Stats in a nice box format
	totalBadge := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true).
		Render(fmt.Sprintf("%d sessions", len(group.Sessions)))
	b.WriteString(totalBadge)
	b.WriteString("\n")

	// Status breakdown with badges
	if running > 0 {
		badge := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Padding(0, 1).
			Render(fmt.Sprintf("● %d running", running))
		b.WriteString(badge)
		b.WriteString(" ")
	}
	if waiting > 0 {
		badge := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorYellow).
			Padding(0, 1).
			Render(fmt.Sprintf("◐ %d waiting", waiting))
		b.WriteString(badge)
		b.WriteString(" ")
	}
	if idle > 0 {
		badge := lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBorder).
			Padding(0, 1).
			Render(fmt.Sprintf("○ %d idle", idle))
		b.WriteString(badge)
		b.WriteString(" ")
	}
	if errored > 0 {
		badge := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 1).
			Render(fmt.Sprintf("✕ %d error", errored))
		b.WriteString(badge)
	}
	b.WriteString("\n\n")

	// Session list header
	listHeader := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("─── Sessions ───")
	b.WriteString(listHeader)
	b.WriteString("\n")

	// Session list
	for i, sess := range group.Sessions {
		if i >= height-10 { // Leave room
			b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d more", len(group.Sessions)-i)))
			break
		}
		statusIcon := "○"
		statusColor := ColorTextDim
		switch sess.Status {
		case session.StatusRunning:
			statusIcon = "●"
			statusColor = ColorGreen
		case session.StatusWaiting:
			statusIcon = "◐"
			statusColor = ColorYellow
		case session.StatusError:
			statusIcon = "✕"
			statusColor = ColorRed
		}
		status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
		name := lipgloss.NewStyle().Foreground(ColorText).Render(sess.Title)
		tool := lipgloss.NewStyle().Foreground(ColorPurple).Render(fmt.Sprintf("[%s]", sess.Tool))
		b.WriteString(fmt.Sprintf("  %s %s %s\n", status, name, tool))
	}

	// Help hint at bottom
	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	b.WriteString(hintStyle.Render("Tab: expand/collapse • G: rename • d: delete"))

	return b.String()
}
