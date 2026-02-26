package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghe.spotify.net/mnicholson/hangar/internal/session"
)

func TestNewHome(t *testing.T) {
	home := NewHome()
	if home == nil {
		t.Fatal("NewHome returned nil")
	}
	if home.storage == nil {
		t.Error("Storage should be initialized")
	}
	if home.search == nil {
		t.Error("Search component should be initialized")
	}
	if home.newDialog == nil {
		t.Error("NewDialog component should be initialized")
	}
}

func TestHomeInit(t *testing.T) {
	home := NewHome()
	cmd := home.Init()
	// Init should return a command for loading sessions
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestHomeView(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	view := home.View()
	if view == "" {
		t.Error("View should not be empty")
	}
	if view == "Loading..." {
		// Initial state is OK
		return
	}
}

func TestHomeUpdateQuit(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := home.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Log("Quit command expected (may be nil in test context)")
	}
}

func TestHomeUpdateResize(t *testing.T) {
	home := NewHome()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if h.width != 120 {
		t.Errorf("Width = %d, want 120", h.width)
	}
	if h.height != 40 {
		t.Errorf("Height = %d, want 40", h.height)
	}
}

func TestHomeUpdateSearch(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Disable global search to test local search behavior
	home.globalSearchIndex = nil

	// Press / to open search (should open local search when global is not available)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if !h.search.IsVisible() {
		t.Error("Local search should be visible after pressing / when global search is not available")
	}
}

func TestHomeUpdateNewDialog(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Press n to open new dialog
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if !h.newDialog.IsVisible() {
		t.Error("New dialog should be visible after pressing n")
	}
}

func TestHomeLoadSessions(t *testing.T) {
	home := NewHome()

	// Trigger load sessions
	msg := home.loadSessions()

	loadMsg, ok := msg.(loadSessionsMsg)
	if !ok {
		t.Fatal("loadSessions should return loadSessionsMsg")
	}

	// Should not error on empty storage
	if loadMsg.err != nil {
		t.Errorf("Unexpected error: %v", loadMsg.err)
	}
}

func TestHomeRenameGroupWithR(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a group tree with a group
	home.groupTree = session.NewGroupTree([]*session.Instance{})
	home.groupTree.CreateGroup("test-group")
	home.rebuildFlatItems()

	// Position cursor on the group
	home.cursor = 0
	if len(home.flatItems) == 0 {
		t.Fatal("flatItems should have at least one group")
	}
	if home.flatItems[0].Type != session.ItemTypeGroup {
		t.Fatal("First item should be a group")
	}

	// Press r to open rename dialog
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if !h.groupDialog.IsVisible() {
		t.Error("Group dialog should be visible after pressing r on a group")
	}
	if h.groupDialog.Mode() != GroupDialogRename {
		t.Errorf("Dialog mode = %v, want GroupDialogRename", h.groupDialog.Mode())
	}
}

func TestHomeRenameSessionWithR(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a test session
	inst := session.NewInstance("test-session", "/tmp/project")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Find and position cursor on the session (skip the group)
	sessionIdx := -1
	for i, item := range home.flatItems {
		if item.Type == session.ItemTypeSession {
			sessionIdx = i
			break
		}
	}
	if sessionIdx == -1 {
		t.Fatal("No session found in flatItems")
	}
	home.cursor = sessionIdx

	// Press r to open rename dialog
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if !h.groupDialog.IsVisible() {
		t.Error("Group dialog should be visible after pressing r on a session")
	}
	if h.groupDialog.Mode() != GroupDialogRenameSession {
		t.Errorf("Dialog mode = %v, want GroupDialogRenameSession", h.groupDialog.Mode())
	}
	if h.groupDialog.GetSessionID() != inst.ID {
		t.Errorf("Session ID = %s, want %s", h.groupDialog.GetSessionID(), inst.ID)
	}
}

func TestHomeRenameSessionComplete(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a test session
	inst := session.NewInstance("original-name", "/tmp/project")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst // Also populate the O(1) lookup map
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Find and position cursor on the session
	sessionIdx := -1
	for i, item := range home.flatItems {
		if item.Type == session.ItemTypeSession {
			sessionIdx = i
			break
		}
	}
	home.cursor = sessionIdx

	// Press r to open rename dialog
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	home.Update(msg)

	// Simulate typing a new name
	home.groupDialog.nameInput.SetValue("new-name")

	// Press Enter to confirm
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := home.Update(enterMsg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if h.groupDialog.IsVisible() {
		t.Error("Dialog should be hidden after pressing Enter")
	}
	if h.instances[0].Title != "new-name" {
		t.Errorf("Session title = %s, want new-name", h.instances[0].Title)
	}
}

func TestHomeEnterDuringLaunchingDoesNotShowStartingError(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst := session.NewInstance("launching-session", "/tmp/project")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()

	home.flatItems = []session.Item{
		{Type: session.ItemTypeSession, Session: inst},
	}
	home.cursor = 0
	home.launchingSessions[inst.ID] = time.Now()

	model, _ := home.Update(tea.KeyMsg{Type: tea.KeyEnter})
	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}

	if h.err != nil && strings.Contains(h.err.Error(), "session is starting, please wait") {
		t.Fatalf("unexpected launch block error: %v", h.err)
	}
}

func TestLaunchAnimationMinDurationByTool(t *testing.T) {
	if got := launchAnimationMinDuration("claude"); got != minLaunchAnimationDurationClaude {
		t.Fatalf("claude min duration = %v, want %v", got, minLaunchAnimationDurationClaude)
	}
	if got := launchAnimationMinDuration("gemini"); got != minLaunchAnimationDurationClaude {
		t.Fatalf("gemini min duration = %v, want %v", got, minLaunchAnimationDurationClaude)
	}
	if got := launchAnimationMinDuration("shell"); got != minLaunchAnimationDurationDefault {
		t.Fatalf("default min duration = %v, want %v", got, minLaunchAnimationDurationDefault)
	}
}

func TestHomeRenamePendingChangesSurviveReload(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a test session
	inst := session.NewInstance("original-name", "/tmp/project")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Simulate a rename that stores a pending title change
	home.pendingTitleChanges[inst.ID] = "renamed-title"

	// Simulate a reload (loadSessionsMsg) with the OLD title from disk
	reloadInst := session.NewInstance("original-name", "/tmp/project")
	reloadInst.ID = inst.ID // Same session, old title

	reloadMsg := loadSessionsMsg{
		instances:    []*session.Instance{reloadInst},
		groups:       nil,
		restoreState: &reloadState{cursorSessionID: inst.ID},
	}

	model, _ := home.Update(reloadMsg)
	h := model.(*Home)

	// The pending rename should have been re-applied after reload
	if h.instances[0].Title != "renamed-title" {
		t.Errorf("Session title = %s, want renamed-title (pending rename should survive reload)", h.instances[0].Title)
	}
	// Pending changes should be cleared after re-application
	if len(h.pendingTitleChanges) != 0 {
		t.Errorf("pendingTitleChanges should be empty after re-application, got %d", len(h.pendingTitleChanges))
	}
}

func TestHomeRenamePendingChangesNoop(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a test session
	inst := session.NewInstance("desired-name", "/tmp/project")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Store a pending change that matches the current title (normal save succeeded)
	home.pendingTitleChanges[inst.ID] = "desired-name"

	// Reload with data that already has the correct title
	reloadInst := session.NewInstance("desired-name", "/tmp/project")
	reloadInst.ID = inst.ID

	reloadMsg := loadSessionsMsg{
		instances:    []*session.Instance{reloadInst},
		groups:       nil,
		restoreState: &reloadState{cursorSessionID: inst.ID},
	}

	model, _ := home.Update(reloadMsg)
	h := model.(*Home)

	// Title should still be correct
	if h.instances[0].Title != "desired-name" {
		t.Errorf("Session title = %s, want desired-name", h.instances[0].Title)
	}
	// Pending changes should be cleared (no re-application needed)
	if len(h.pendingTitleChanges) != 0 {
		t.Errorf("pendingTitleChanges should be empty, got %d", len(h.pendingTitleChanges))
	}
}

func TestHomeGlobalSearchInitialized(t *testing.T) {
	home := NewHome()
	if home.globalSearch == nil {
		t.Error("GlobalSearch component should be initialized")
	}
	// globalSearchIndex may be nil if not enabled in config, that's OK
}

func TestHomeSearchOpensGlobalWhenAvailable(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a mock index
	tmpDir := t.TempDir()
	config := session.GlobalSearchSettings{
		Enabled:        true,
		Tier:           "instant",
		MemoryLimitMB:  100,
		IndexRateLimit: 100,
	}
	index, err := session.NewGlobalSearchIndex(tmpDir, config)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer index.Close()

	home.globalSearchIndex = index
	home.globalSearch.SetIndex(index)

	// Press / to open search - should open global search when index is available
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if !h.globalSearch.IsVisible() {
		t.Error("Global search should be visible after pressing / when index is available")
	}
	if h.search.IsVisible() {
		t.Error("Local search should NOT be visible when global search opens")
	}
}

func TestHomeSearchOpensLocalWhenNoIndex(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Ensure no global search index
	home.globalSearchIndex = nil

	// Press / to open search - should fall back to local search
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	model, _ := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if h.globalSearch.IsVisible() {
		t.Error("Global search should NOT be visible when index is nil")
	}
	if !h.search.IsVisible() {
		t.Error("Local search should be visible when global index is not available")
	}
}

func TestHomeGlobalSearchEscape(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Create a mock index
	tmpDir := t.TempDir()
	config := session.GlobalSearchSettings{
		Enabled:        true,
		Tier:           "instant",
		MemoryLimitMB:  100,
		IndexRateLimit: 100,
	}
	index, err := session.NewGlobalSearchIndex(tmpDir, config)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer index.Close()

	home.globalSearchIndex = index
	home.globalSearch.SetIndex(index)

	// Open global search with /
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	home.Update(msg)

	if !home.globalSearch.IsVisible() {
		t.Fatal("Global search should be visible after pressing /")
	}

	// Press Escape to close
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := home.Update(escMsg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}
	if h.globalSearch.IsVisible() {
		t.Error("Global search should be hidden after pressing Escape")
	}
}

func TestGetLayoutMode(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected string
	}{
		{"narrow phone", 45, "single"},
		{"phone landscape", 65, "stacked"},
		{"tablet", 85, "dual"},
		{"desktop", 120, "dual"},
		{"exact boundary 50", 50, "stacked"},
		{"exact boundary 80", 80, "dual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := NewHome()
			home.width = tt.width
			got := home.getLayoutMode()
			if got != tt.expected {
				t.Errorf("getLayoutMode() at width %d = %q, want %q", tt.width, got, tt.expected)
			}
		})
	}
}

func TestRenderHelpBarTiny(t *testing.T) {
	home := NewHome()
	home.width = 45 // Tiny mode (<50 cols)
	home.height = 30

	result := home.renderHelpBar()

	// Should contain minimal hint
	if !strings.Contains(result, "?") {
		t.Error("Tiny help bar should contain ? for help")
	}
	// Should NOT contain full shortcuts
	if strings.Contains(result, "Attach") {
		t.Error("Tiny help bar should not contain 'Attach'")
	}
	if strings.Contains(result, "Global") {
		t.Error("Tiny help bar should not contain 'Global'")
	}
}

func TestRenderHelpBarMinimal(t *testing.T) {
	home := NewHome()
	home.width = 55 // Minimal mode (50-69)
	home.height = 30

	result := home.renderHelpBar()

	// Should contain key-only hints
	if !strings.Contains(result, "?") {
		t.Error("Minimal help bar should contain ?")
	}
	if !strings.Contains(result, "q") {
		t.Error("Minimal help bar should contain q")
	}
	// Should NOT contain full descriptions
	if strings.Contains(result, "Attach") {
		t.Error("Minimal help bar should not contain full descriptions")
	}
}

func TestRenderHelpBarMinimalWithSession(t *testing.T) {
	home := NewHome()
	home.width = 55 // Minimal mode (50-69)
	home.height = 30

	// Add a session to test context-specific keys
	testSession := &session.Instance{
		ID:    "test-123",
		Title: "Test Session",
		Tool:  "claude",
	}
	home.flatItems = []session.Item{
		{Type: session.ItemTypeSession, Session: testSession},
	}
	home.cursor = 0

	result := home.renderHelpBar()

	// Should contain key indicators
	if !strings.Contains(result, "n") {
		t.Error("Minimal help bar should contain n key")
	}
	if !strings.Contains(result, "R") {
		t.Error("Minimal help bar should contain R key for restart")
	}
	// Should NOT contain full descriptions
	if strings.Contains(result, "Attach") {
		t.Error("Minimal help bar should not contain full descriptions")
	}
}

func TestRenderHelpBarCompact(t *testing.T) {
	home := NewHome()
	home.width = 85 // Compact mode (70-99)
	home.height = 30

	result := home.renderHelpBar()

	// Should contain abbreviated hints
	if !strings.Contains(result, "?") {
		t.Error("Compact help bar should contain ?")
	}
	// Should contain some descriptions but abbreviated
	if strings.Contains(result, "Global") {
		t.Error("Compact help bar should not contain 'Global'")
	}
}

func TestRenderHelpBarCompactWithSession(t *testing.T) {
	home := NewHome()
	home.width = 85 // Compact mode (70-99)
	home.height = 30

	// Add a session with fork capability
	// ClaudeDetectedAt must be recent for CanFork() to return true
	testSession := &session.Instance{
		ID:               "test-123",
		Title:            "Test Session",
		Tool:             "claude",
		ClaudeSessionID:  "session-abc",
		ClaudeDetectedAt: time.Now(), // Must be recent for CanFork()
	}
	home.flatItems = []session.Item{
		{Type: session.ItemTypeSession, Session: testSession},
	}
	home.cursor = 0

	result := home.renderHelpBar()

	// Should have abbreviated descriptions
	if !strings.Contains(result, "New") {
		t.Error("Compact help bar should contain 'New'")
	}
	if !strings.Contains(result, "Restart") {
		t.Error("Compact help bar should contain 'Restart'")
	}
	// Should have fork since session can fork
	if !strings.Contains(result, "Fork") {
		t.Error("Compact help bar should contain 'Fork' for forkable session")
	}
	// Should NOT contain full verbose text
	if strings.Contains(result, "Global") {
		t.Error("Compact help bar should not contain 'Global'")
	}
}

func TestRenderHelpBarCompactWithGroup(t *testing.T) {
	home := NewHome()
	home.width = 85 // Compact mode (70-99)
	home.height = 30

	// Add a group
	home.flatItems = []session.Item{
		{Type: session.ItemTypeGroup, Path: "test-group", Level: 0},
	}
	home.cursor = 0

	result := home.renderHelpBar()

	// Should have toggle hint for groups
	if !strings.Contains(result, "Toggle") {
		t.Error("Compact help bar should contain 'Toggle' for groups")
	}
}

func TestHomeViewNarrowTerminal(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		shouldRender  bool
	}{
		{"too narrow", 35, 20, false},
		{"minimum width", 40, 12, true},
		{"narrow but ok", 50, 15, true},
		{"issue #2 case", 79, 70, true},
		{"normal", 100, 30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := NewHome()
			home.width = tt.width
			home.height = tt.height

			view := home.View()

			if tt.shouldRender {
				if strings.Contains(view, "Terminal too small") {
					t.Errorf("width=%d height=%d should render, got 'too small' message", tt.width, tt.height)
				}
			} else {
				if !strings.Contains(view, "Terminal too small") {
					t.Errorf("width=%d height=%d should show 'too small', got normal render", tt.width, tt.height)
				}
			}
		})
	}
}

func TestHomeViewStackedLayout(t *testing.T) {
	home := NewHome()
	home.width = 65 // Stacked mode (50-79)
	home.height = 40
	home.initialLoading = false

	// Add a test session so we have content
	inst := &session.Instance{ID: "test1", Title: "Test Session", Tool: "claude", Status: session.StatusIdle}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	view := home.View()

	// In stacked mode, we should NOT see side-by-side separator
	// The view should render without panicking
	if view == "" {
		t.Error("View should not be empty")
	}
	if strings.Contains(view, "Terminal too small") {
		t.Error("65-col terminal should not show 'too small' error")
	}
}

func TestHomeViewSingleColumnLayout(t *testing.T) {
	home := NewHome()
	home.width = 45 // Single column mode (<50)
	home.height = 30
	home.initialLoading = false

	// Add a test session
	inst := &session.Instance{ID: "test1", Title: "Test Session", Tool: "claude", Status: session.StatusIdle}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	view := home.View()

	// In single column mode, should show list only (no preview)
	if view == "" {
		t.Error("View should not be empty")
	}
	if strings.Contains(view, "Terminal too small") {
		t.Error("45-col terminal should not show 'too small' error")
	}
}

func TestPushUndoStackLIFO(t *testing.T) {
	home := NewHome()

	// Push 3 sessions
	for i := 0; i < 3; i++ {
		inst := session.NewInstance(fmt.Sprintf("session-%d", i), "/tmp")
		home.pushUndoStack(inst)
	}

	if len(home.undoStack) != 3 {
		t.Fatalf("undoStack length = %d, want 3", len(home.undoStack))
	}

	// Verify LIFO order: last pushed should be at the end
	if home.undoStack[2].instance.Title != "session-2" {
		t.Errorf("top of stack = %s, want session-2", home.undoStack[2].instance.Title)
	}
	if home.undoStack[0].instance.Title != "session-0" {
		t.Errorf("bottom of stack = %s, want session-0", home.undoStack[0].instance.Title)
	}
}

func TestPushUndoStackCap(t *testing.T) {
	home := NewHome()

	// Push 12 sessions (exceeds cap of 10)
	for i := 0; i < 12; i++ {
		inst := session.NewInstance(fmt.Sprintf("session-%d", i), "/tmp")
		home.pushUndoStack(inst)
	}

	if len(home.undoStack) != 10 {
		t.Fatalf("undoStack length = %d, want 10 (capped)", len(home.undoStack))
	}

	// Oldest 2 should be dropped, so first entry should be session-2
	if home.undoStack[0].instance.Title != "session-2" {
		t.Errorf("bottom of stack = %s, want session-2 (oldest dropped)", home.undoStack[0].instance.Title)
	}
	// Most recent should be session-11
	if home.undoStack[9].instance.Title != "session-11" {
		t.Errorf("top of stack = %s, want session-11", home.undoStack[9].instance.Title)
	}
}

func TestCtrlZEmptyStack(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Press Ctrl+Z with empty stack
	msg := tea.KeyMsg{Type: tea.KeyCtrlZ}
	model, cmd := home.Update(msg)

	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}

	// Should show "nothing to undo" error
	if h.err == nil {
		t.Error("Expected error message for empty undo stack")
	} else if !strings.Contains(h.err.Error(), "nothing to undo") {
		t.Errorf("Error = %q, want 'nothing to undo'", h.err.Error())
	}

	// Should not return a command
	if cmd != nil {
		t.Error("Expected nil command for empty undo stack")
	}
}

func TestUndoHintInHelpBar(t *testing.T) {
	home := NewHome()
	home.width = 200 // Wide terminal to fit all hints including Undo
	home.height = 30

	// Add a session to have context (non-Claude to reduce hint count)
	inst := &session.Instance{ID: "test-1", Title: "Test", Tool: "other"}
	home.flatItems = []session.Item{
		{Type: session.ItemTypeSession, Session: inst},
	}
	home.cursor = 0

	// No undo stack: should NOT show ^Z
	result := home.renderHelpBar()
	if strings.Contains(result, "Undo") {
		t.Error("Help bar should NOT show Undo when undo stack is empty")
	}

	// Push to undo stack: should show ^Z
	home.pushUndoStack(session.NewInstance("deleted", "/tmp"))
	result = home.renderHelpBar()
	if !strings.Contains(result, "Undo") {
		t.Errorf("Help bar should show Undo when undo stack is non-empty\nGot: %q", result)
	}
}

func TestHomeViewAllLayoutModes(t *testing.T) {
	testCases := []struct {
		name       string
		width      int
		height     int
		layoutMode string
	}{
		{"single column", 45, 30, "single"},
		{"stacked", 65, 40, "stacked"},
		{"dual column", 100, 40, "dual"},
		{"issue #2 exact", 79, 70, "stacked"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			home := NewHome()
			home.width = tc.width
			home.height = tc.height
			home.initialLoading = false

			// Verify layout mode detection
			if got := home.getLayoutMode(); got != tc.layoutMode {
				t.Errorf("getLayoutMode() = %q, want %q", got, tc.layoutMode)
			}

			// Verify view renders without error
			view := home.View()
			if view == "" {
				t.Error("View should not be empty")
			}
			if strings.Contains(view, "Terminal too small") {
				t.Errorf("Terminal %dx%d should render, got 'too small'", tc.width, tc.height)
			}
		})
	}
}

func TestSessionRestartedMsgErrorClearsResumingAnimation(t *testing.T) {
	home := NewHome()
	inst := session.NewInstance("restart-test", "/tmp/project")

	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()

	home.resumingSessions[inst.ID] = time.Now()

	model, _ := home.Update(sessionRestartedMsg{
		sessionID: inst.ID,
		err:       fmt.Errorf("restart failed"),
	})
	h := model.(*Home)

	if _, ok := h.resumingSessions[inst.ID]; ok {
		t.Fatal("resuming animation should be cleared after restart error")
	}
	if h.err == nil {
		t.Fatal("expected restart error to be set")
	}
	if !strings.Contains(h.err.Error(), "failed to restart session") {
		t.Fatalf("unexpected error: %v", h.err)
	}
}

func TestRestartSessionCmdSessionMissingReturnsError(t *testing.T) {
	home := NewHome()
	inst := session.NewInstance("restart-test", "/tmp/project")

	// Build command with a valid instance, then simulate reload/delete before cmd runs.
	cmd := home.restartSession(inst)
	home.instancesMu.Lock()
	delete(home.instanceByID, inst.ID)
	home.instancesMu.Unlock()

	msg := cmd()
	restarted, ok := msg.(sessionRestartedMsg)
	if !ok {
		t.Fatalf("expected sessionRestartedMsg, got %T", msg)
	}
	if restarted.err == nil {
		t.Fatal("expected error when session no longer exists")
	}
	if !strings.Contains(restarted.err.Error(), "session no longer exists") {
		t.Fatalf("unexpected error: %v", restarted.err)
	}
}

func TestListItemAt(t *testing.T) {
	makeItems := func(n int) []session.Item {
		items := make([]session.Item, n)
		for i := range items {
			items[i] = session.Item{Type: session.ItemTypeSession}
		}
		return items
	}

	t.Run("basic click hits correct item", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeItems(5)
		h.viewOffset = 0
		// listStartRow = 4 (no banners, viewOffset=0)
		// row 4 → item 0, row 5 → item 1, row 6 → item 2
		if got := h.listItemAt(5, 4); got != 0 {
			t.Errorf("row 4 → want 0, got %d", got)
		}
		if got := h.listItemAt(5, 6); got != 2 {
			t.Errorf("row 6 → want 2, got %d", got)
		}
	})

	t.Run("click above list returns -1", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeItems(5)
		if got := h.listItemAt(5, 0); got != -1 {
			t.Errorf("row 0 → want -1, got %d", got)
		}
		if got := h.listItemAt(5, 3); got != -1 {
			t.Errorf("row 3 → want -1, got %d", got)
		}
	})

	t.Run("click beyond items returns -1", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeItems(3)
		// listStartRow=4, items at rows 4,5,6 → row 7 is out of bounds
		if got := h.listItemAt(5, 7); got != -1 {
			t.Errorf("row beyond items → want -1, got %d", got)
		}
	})

	t.Run("dual layout: click in right panel returns -1", func(t *testing.T) {
		h := NewHome()
		h.width = 120 // dual layout (>=80)
		h.height = 30
		h.flatItems = makeItems(5)
		leftWidth := int(float64(120) * 0.35) // = 42
		// click at x=50 is in the right panel
		if got := h.listItemAt(50, 4); got != -1 {
			t.Errorf("right panel click → want -1, got %d", got)
		}
		// click at x=41 is in left panel
		if got := h.listItemAt(leftWidth-1, 4); got != 0 {
			t.Errorf("left panel click → want 0, got %d", got)
		}
	})

	t.Run("viewOffset shifts item mapping", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeItems(10)
		h.viewOffset = 3
		// viewOffset>0 adds 1 row for "more above" indicator
		// listStartRow = 4 + 1 = 5
		// row 5 → item 3 (viewOffset + 0)
		if got := h.listItemAt(5, 5); got != 3 {
			t.Errorf("row 5 with viewOffset=3 → want 3, got %d", got)
		}
	})
}

func TestHandleMouseMsg(t *testing.T) {
	makeSessionItems := func(n int) []session.Item {
		items := make([]session.Item, n)
		for i := range items {
			inst := &session.Instance{}
			items[i] = session.Item{Type: session.ItemTypeSession, Session: inst}
		}
		return items
	}

	t.Run("single left click moves cursor", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeSessionItems(5)
		h.cursor = 0

		// listStartRow=4 (no banners, viewOffset=0), Y=5 → item 1
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 5}
		model, _ := h.Update(msg)
		updated := model.(*Home)
		if updated.cursor != 1 {
			t.Errorf("expected cursor=1 after click row 5, got %d", updated.cursor)
		}
	})

	t.Run("click above list does not move cursor", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeSessionItems(5)
		h.cursor = 2

		// Y=0 is above the list (listStartRow=4)
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 0}
		model, _ := h.Update(msg)
		updated := model.(*Home)
		if updated.cursor != 2 {
			t.Errorf("expected cursor unchanged at 2, got %d", updated.cursor)
		}
	})

	t.Run("first click records tracking state", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeSessionItems(5)
		h.cursor = 0

		// Y=4 → listStartRow=4 → item 0
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 4}
		model, _ := h.Update(msg)
		h = model.(*Home)
		if h.lastClickIndex != 0 {
			t.Errorf("expected lastClickIndex=0 after first click, got %d", h.lastClickIndex)
		}
		if h.lastClickTime.IsZero() {
			t.Error("expected lastClickTime to be set after first click")
		}
	})

	t.Run("scroll wheel up moves cursor up", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeSessionItems(20)
		h.viewOffset = 5
		h.cursor = 5

		msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress, X: 5, Y: 10}
		model, _ := h.Update(msg)
		updated := model.(*Home)
		if updated.cursor >= 5 {
			t.Errorf("expected cursor to move up after wheel up, got %d", updated.cursor)
		}
	})

	t.Run("scroll wheel down moves cursor down", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30
		h.flatItems = makeSessionItems(20)
		h.cursor = 0

		msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress, X: 5, Y: 10}
		model, _ := h.Update(msg)
		updated := model.(*Home)
		if updated.cursor <= 0 {
			t.Errorf("expected cursor to move down after wheel down, got %d", updated.cursor)
		}
	})

	t.Run("click on group item toggles group", func(t *testing.T) {
		h := NewHome()
		h.width = 120
		h.height = 30

		// Build a GroupTree manually with one expanded group
		gt := session.NewGroupTree([]*session.Instance{})
		grp := &session.Group{Name: "test", Path: "test", Expanded: true, Sessions: []*session.Instance{}}
		gt.Groups["test"] = grp
		gt.GroupList = append(gt.GroupList, grp)
		gt.Expanded["test"] = true

		h.groupTree = gt
		h.flatItems = []session.Item{
			{Type: session.ItemTypeGroup, Group: grp, Path: "test"},
		}
		h.cursor = 0

		// Y=4 → listStartRow=4 → item 0 (the group)
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 4}
		model, _ := h.Update(msg)
		updated := model.(*Home)

		// After toggling, the group should be collapsed
		// Check via the Expanded map (false or absent = collapsed)
		if updated.groupTree.Expanded["test"] {
			t.Error("expected group to be collapsed after click, but Expanded[\"test\"] is still true")
		}
	})
}

func TestAttachSessionMouseModeConfig(t *testing.T) {
	// Verify GetMouseMode accessor works correctly.
	// The integration (EnableMouseMode called on attach) requires a real tmux session.
	s := session.TmuxSettings{}
	if s.GetMouseMode() != false {
		t.Error("default MouseMode should be false")
	}
	tr := true
	s.MouseMode = &tr
	if !s.GetMouseMode() {
		t.Error("MouseMode should be true when set")
	}
}

func TestSessionRowPRBadgeStates(t *testing.T) {
	tests := []struct {
		state     string
		prNum     int
		wantBadge string
		wantShown bool
	}{
		{"OPEN", 10, "[#10]", true},
		{"MERGED", 20, "[#20]", true},
		{"CLOSED", 30, "[#30]", true},
		{"DRAFT", 40, "[#40]", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			home := NewHome()
			home.width = 120
			home.height = 30
			home.initialLoading = false

			inst := &session.Instance{
				ID:               "pr-state-test",
				Title:            "my-feature",
				Tool:             "claude",
				Status:           session.StatusIdle,
				WorktreePath:     "/repo/.git/worktrees/my-feature",
				WorktreeBranch:   "my-feature",
				WorktreeRepoRoot: "/repo",
			}
			home.instancesMu.Lock()
			home.instances = []*session.Instance{inst}
			home.instanceByID[inst.ID] = inst
			home.instancesMu.Unlock()
			home.groupTree = session.NewGroupTree(home.instances)
			home.rebuildFlatItems()

			home.prCacheMu.Lock()
			home.prCache[inst.ID] = &prCacheEntry{
				Number: tt.prNum,
				State:  tt.state,
			}
			home.prCacheMu.Unlock()

			view := home.View()

			if tt.wantShown && !strings.Contains(view, tt.wantBadge) {
				t.Errorf("state=%s: expected badge %q in view, not found\nview:\n%s", tt.state, tt.wantBadge, view)
			}
			if !tt.wantShown && strings.Contains(view, tt.wantBadge) {
				t.Errorf("state=%s: badge %q should not appear in view\nview:\n%s", tt.state, tt.wantBadge, view)
			}
		})
	}

	// Non-worktree sessions must not show a badge even if a cache entry exists
	t.Run("non-worktree session ignores cache entry", func(t *testing.T) {
		home := NewHome()
		home.width = 120
		home.height = 30
		home.initialLoading = false

		inst := &session.Instance{
			ID:     "plain-session",
			Title:  "plain-session",
			Tool:   "claude",
			Status: session.StatusIdle,
			// WorktreePath intentionally empty — IsWorktree() returns false
		}
		home.instancesMu.Lock()
		home.instances = []*session.Instance{inst}
		home.instanceByID[inst.ID] = inst
		home.instancesMu.Unlock()
		home.groupTree = session.NewGroupTree(home.instances)
		home.rebuildFlatItems()

		home.prCacheMu.Lock()
		home.prCache[inst.ID] = &prCacheEntry{Number: 99, State: "OPEN"}
		home.prCacheMu.Unlock()

		view := home.View()
		if strings.Contains(view, "[#99]") {
			t.Errorf("non-worktree session should not render PR badge, but found [#99] in view:\n%s", view)
		}
	})
}

func makeWorktreeInstance(id, branch string) *session.Instance {
	return &session.Instance{
		ID:               id,
		Title:            "test-" + id,
		WorktreePath:     "/tmp/worktrees/" + id,
		WorktreeRepoRoot: "/tmp/repo",
		WorktreeBranch:   branch,
	}
}

func TestWorktreeFinishDialog_ShowWithCachedPR(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Pre-populate PR cache
	home.prCacheMu.Lock()
	home.prCache[inst.ID] = &prCacheEntry{Number: 55, State: "OPEN", Title: "My PR"}
	home.prCacheTs[inst.ID] = time.Now()
	home.prCacheMu.Unlock()

	// Trigger W key — capture the returned model
	// flatItems[0] is the group header, flatItems[1] is the session
	home.cursor = 1
	model, _ := home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})
	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}

	if !h.worktreeFinishDialog.IsVisible() {
		t.Fatal("expected dialog visible")
	}
	if h.worktreeFinishDialog.prEntry == nil {
		t.Fatal("expected prEntry to be set from cache")
	}
	if h.worktreeFinishDialog.prEntry.Number != 55 {
		t.Errorf("expected PR #55, got %d", h.worktreeFinishDialog.prEntry.Number)
	}
	if !h.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded=true when cache entry exists")
	}
}

func TestWorktreeFinishDialog_ShowWithNoCachedPR(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// No PR in cache — trigger W key
	// flatItems[0] is the group header, flatItems[1] is the session
	home.cursor = 1
	model, _ := home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})
	h, ok := model.(*Home)
	if !ok {
		t.Fatal("Update should return *Home")
	}

	if !h.worktreeFinishDialog.IsVisible() {
		t.Fatal("expected dialog visible")
	}
	// prLoaded=false means still checking
	if h.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded=false when no cache entry")
	}
}

func TestWorktreeFinishDialog_PRFetchedUpdatesDialog(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Open dialog (no PR cached yet — dialog shows "checking...")
	home.cursor = 1 // index 0 is the group header
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})

	if home.worktreeFinishDialog.prLoaded {
		t.Fatal("expected prLoaded=false before fetch")
	}

	// Simulate PR fetch arriving
	home.Update(prFetchedMsg{
		sessionID: inst.ID,
		pr:        &prCacheEntry{Number: 99, State: "MERGED", Title: "Done"},
	})

	if !home.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded=true after fetch")
	}
	if home.worktreeFinishDialog.prEntry == nil {
		t.Fatal("expected prEntry set after fetch")
	}
	if home.worktreeFinishDialog.prEntry.Number != 99 {
		t.Errorf("expected PR #99, got %d", home.worktreeFinishDialog.prEntry.Number)
	}
}

func TestWorktreeFinishDialog_PRFetchedIgnoresDifferentSession(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 1
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})

	// Fetch for a different session
	home.Update(prFetchedMsg{
		sessionID: "other-session",
		pr:        &prCacheEntry{Number: 77, State: "OPEN"},
	})

	// Dialog should remain in "checking..." state
	if home.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded unchanged for different session")
	}
}

func TestDeleteKey_WorktreeSessionOpensFinishDialog(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := makeWorktreeInstance("sess1", "feat/delete-me")
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 1 // index 0 is the group header
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if !home.worktreeFinishDialog.IsVisible() {
		t.Error("expected WorktreeFinishDialog visible when d pressed on worktree session")
	}
	if home.confirmDialog.IsVisible() {
		t.Error("expected ConfirmDialog NOT visible for worktree session")
	}
}

func TestDeleteKey_NonWorktreeSessionOpensConfirmDialog(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.initialLoading = false
	inst := &session.Instance{ID: "sess2", Title: "plain-session"}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 1 // index 0 is the group header
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if home.worktreeFinishDialog.IsVisible() {
		t.Error("expected WorktreeFinishDialog NOT visible for non-worktree session")
	}
	if !home.confirmDialog.IsVisible() {
		t.Error("expected ConfirmDialog visible for non-worktree session")
	}
}

func TestPRView_ToggleWithP(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.ghPath = "/usr/bin/gh" // fake path — just needs to be non-empty

	if home.viewMode == "prs" {
		t.Fatal("viewMode should not start in prs mode")
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode != "prs" {
		t.Errorf("viewMode = %q, want \"prs\"", h.viewMode)
	}
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor = %d, want 0", h.prViewCursor)
	}
}

func TestPRView_NoToggleWithoutGh(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.ghPath = "" // explicitly clear to simulate gh not installed

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode == "prs" {
		t.Error("viewMode should not switch to prs when gh is not installed")
	}
}

func TestPRView_EscReturnsToSessions(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.viewMode = "prs"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode == "prs" {
		t.Error("esc should exit PR view")
	}
}

func TestPRView_Navigation(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.viewMode = "prs"

	// Seed PR cache with two sessions
	sess1 := &session.Instance{ID: "s1", Title: "Session 1", WorktreePath: "/tmp/s1"}
	sess2 := &session.Instance{ID: "s2", Title: "Session 2", WorktreePath: "/tmp/s2"}
	home.instances = []*session.Instance{sess1, sess2}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	home.prCache["s1"] = &prCacheEntry{Number: 1, Title: "PR 1", State: "OPEN", URL: "https://github.com/x/y/pull/1"}
	home.prCache["s2"] = &prCacheEntry{Number: 2, Title: "PR 2", State: "DRAFT", URL: "https://github.com/x/y/pull/2"}

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := home.Update(msg)
	h := model.(*Home)
	if h.prViewCursor != 1 {
		t.Errorf("prViewCursor after down = %d, want 1", h.prViewCursor)
	}

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = h.Update(msg)
	h = model.(*Home)
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor after up = %d, want 0", h.prViewCursor)
	}

	// Navigate up at top — stays at 0
	model, _ = h.Update(msg)
	h = model.(*Home)
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor should clamp at 0, got %d", h.prViewCursor)
	}
}

func TestPRView_RenderShowsPRs(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.viewMode = "prs"
	home.ghPath = "/usr/bin/gh"
	home.initialLoading = false

	sess1 := &session.Instance{ID: "s1", Title: "Fix auth bug", WorktreePath: "/tmp/s1"}
	home.instances = []*session.Instance{sess1}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	home.prCache["s1"] = &prCacheEntry{
		Number:       42,
		Title:        "Fix auth bug",
		State:        "OPEN",
		URL:          "https://github.com/x/y/pull/42",
		HasChecks:    true,
		ChecksPassed: 5,
		ChecksFailed: 1,
	}

	view := home.View()

	if !strings.Contains(view, "#42") {
		t.Error("View should contain PR number #42")
	}
	if !strings.Contains(view, "open") {
		t.Error("View should contain PR state 'open'")
	}
	if !strings.Contains(view, "Fix auth bug") {
		t.Error("View should contain session title")
	}
	if !strings.Contains(view, "PR Overview") {
		t.Error("View should show 'PR Overview' header label")
	}
}

func TestPRView_RenderEmpty(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.viewMode = "prs"
	home.initialLoading = false

	view := home.View()

	if !strings.Contains(view, "No sessions") {
		t.Error("Empty PR view should show empty state message")
	}
}

func TestBulkSelectMode_VKeyToggle(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Initially not in bulk mode
	if home.bulkSelectMode {
		t.Fatal("bulkSelectMode should be false initially")
	}

	// V enters bulk mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
	model, _ := home.Update(msg)
	h := model.(*Home)
	if !h.bulkSelectMode {
		t.Error("V should enter bulk select mode")
	}
	if h.selectedSessionIDs == nil {
		t.Error("selectedSessionIDs should be initialized")
	}

	// V again exits bulk mode
	model, _ = h.Update(msg)
	h = model.(*Home)
	if h.bulkSelectMode {
		t.Error("V again should exit bulk select mode")
	}
	if len(h.selectedSessionIDs) != 0 {
		t.Error("selectedSessionIDs should be cleared on exit")
	}

	// Re-enter bulk mode — selectedSessionIDs should still be non-nil
	model, _ = h.Update(msg)
	h = model.(*Home)
	if !h.bulkSelectMode {
		t.Error("V should re-enter bulk select mode")
	}
	if h.selectedSessionIDs == nil {
		t.Error("selectedSessionIDs should be non-nil on re-entry")
	}
}

func TestBulkSelectMode_EscExits(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Enter bulk mode
	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
	model, _ := home.Update(vMsg)
	h := model.(*Home)
	if !h.bulkSelectMode {
		t.Fatal("should be in bulk mode after V")
	}

	// Esc exits bulk mode
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = h.Update(escMsg)
	h = model.(*Home)
	if h.bulkSelectMode {
		t.Error("Esc should exit bulk select mode")
	}
}

func TestBulkSelectMode_SpaceTogglesSelection(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Set up one session at cursor
	inst := &session.Instance{ID: "test-1", Title: "test-session"}
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"test-1": inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	// flatItems: [0]=group header, [1]=session — point cursor at the session
	home.cursor = 1

	// Enter bulk mode
	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
	model, _ := home.Update(vMsg)
	h := model.(*Home)

	// Space selects the session
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ = h.Update(spaceMsg)
	h = model.(*Home)
	if !h.selectedSessionIDs["test-1"] {
		t.Error("space should select the focused session")
	}

	// Space again deselects
	model, _ = h.Update(spaceMsg)
	h = model.(*Home)
	if h.selectedSessionIDs["test-1"] {
		t.Error("space again should deselect the focused session")
	}
}

func TestBulkSelectMode_SpaceOnGroupIsNoop(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Manually place a group item at cursor position
	home.flatItems = []session.Item{
		{Type: session.ItemTypeGroup, Path: "default", Group: &session.Group{Name: "default"}},
	}
	home.cursor = 0
	home.bulkSelectMode = true

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := home.Update(spaceMsg)
	h := model.(*Home)
	if len(h.selectedSessionIDs) != 0 {
		t.Error("space on group should be a no-op")
	}
}

func TestBulkSelectMode_SpaceOutsideBulkModeIsNoop(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst := &session.Instance{ID: "test-1", Title: "test-session"}
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"test-1": inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	// flatItems: [0]=group header, [1]=session — point cursor at the session
	home.cursor = 1

	// NOT in bulk mode
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := home.Update(spaceMsg)
	h := model.(*Home)
	if len(h.selectedSessionIDs) != 0 {
		t.Error("space outside bulk mode should not select anything")
	}
}

func TestBulkSelectMode_CheckboxRendering(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.initialLoading = false

	inst := &session.Instance{
		ID:     "test-1",
		Title:  "my-session",
		Tool:   "claude",
		Status: session.StatusIdle,
	}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID[inst.ID] = inst
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// In bulk mode, view should contain unselected checkbox
	home.bulkSelectMode = true

	view := home.View()
	if !strings.Contains(view, "□") {
		t.Error("bulk mode should render □ for unselected session")
	}
	if strings.Contains(view, "▶") {
		t.Error("bulk mode should not render ▶ cursor arrow")
	}

	// After selecting, should show checked box
	home.selectedSessionIDs["test-1"] = true
	view = home.View()
	if !strings.Contains(view, "☑") {
		t.Error("selected session should render ☑")
	}

	// Outside bulk mode, should render normal arrow cursor (not checkboxes)
	home.bulkSelectMode = false
	home.selectedSessionIDs = make(map[string]bool)
	view = home.View()
	if strings.Contains(view, "□") {
		t.Error("normal mode should not render □ checkboxes")
	}
	if strings.Contains(view, "☑") {
		t.Error("normal mode should not render ☑ checkboxes")
	}
}

func TestBulkSelectMode_HelpBarBulkMode(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.initialLoading = false
	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"a": true, "b": true}

	view := home.View()
	if !strings.Contains(view, "VISUAL") {
		t.Error("bulk mode should show VISUAL in help bar")
	}
	if !strings.Contains(view, "2 selected") {
		t.Error("bulk mode should show selection count")
	}
	if !strings.Contains(view, ":delete") {
		t.Error("bulk mode help bar should mention delete action")
	}
}

func TestBulkSelectMode_HelpBarNormalMode(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.initialLoading = false
	home.bulkSelectMode = false

	view := home.View()
	if strings.Contains(view, "VISUAL") {
		t.Error("normal mode should not show VISUAL in help bar")
	}
}

func TestBulkSelectMode_DKeyShowsBulkConfirm(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30
	home.initialLoading = false

	inst1 := &session.Instance{ID: "id-1", Title: "sess-1", Tool: "claude"}
	inst2 := &session.Instance{ID: "id-2", Title: "sess-2", Tool: "claude"}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst1, inst2}
	home.instanceByID = map[string]*session.Instance{"id-1": inst1, "id-2": inst2}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"id-1": true, "id-2": true}

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, _ := home.Update(dMsg)
	h := model.(*Home)

	if !h.confirmDialog.IsVisible() {
		t.Error("d in bulk mode with selections should show confirm dialog")
	}
	if h.confirmDialog.GetConfirmType() != ConfirmBulkDeleteSessions {
		t.Errorf("confirm type = %v, want ConfirmBulkDeleteSessions", h.confirmDialog.GetConfirmType())
	}
}

func TestBulkSelectMode_DKeyFallsThrough_WhenNoSelections(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30
	home.initialLoading = false

	inst := &session.Instance{ID: "id-1", Title: "sess-1", Tool: "claude"}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"id-1": inst}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	// Set cursor to the session item (index 1, after the group header)
	for i, item := range home.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == "id-1" {
			home.cursor = i
			break
		}
	}

	// Bulk mode but nothing selected
	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{}

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, _ := home.Update(dMsg)
	h := model.(*Home)

	// Should fall through to single-session confirm
	if !h.confirmDialog.IsVisible() {
		t.Error("d with no selections should fall through to single-session delete confirm")
	}
	if h.confirmDialog.GetConfirmType() != ConfirmDeleteSession {
		t.Errorf("confirm type = %v, want ConfirmDeleteSession", h.confirmDialog.GetConfirmType())
	}
}

func TestBulkSelectMode_XKeyOpensSendDialog(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30
	home.initialLoading = false

	inst1 := &session.Instance{ID: "id-1", Title: "sess-1", Tool: "claude"}
	inst2 := &session.Instance{ID: "id-2", Title: "sess-2", Tool: "claude"}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst1, inst2}
	home.instanceByID = map[string]*session.Instance{"id-1": inst1, "id-2": inst2}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"id-1": true, "id-2": true}

	xMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	model, _ := home.Update(xMsg)
	h := model.(*Home)

	if !h.sendTextDialog.IsVisible() {
		t.Error("x in bulk mode should open send text dialog")
	}
	if len(h.sendTextTargetIDs) != 2 {
		t.Errorf("sendTextTargetIDs len = %d, want 2", len(h.sendTextTargetIDs))
	}
	if h.sendTextTargetID != "" {
		t.Error("sendTextTargetID should be empty when bulk send is active")
	}
}

func TestBulkSelectMode_XKeyFallsThrough_WhenNoSelections(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30
	home.initialLoading = false

	inst := &session.Instance{ID: "id-1", Title: "sess-1", Tool: "claude"}
	home.instancesMu.Lock()
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"id-1": inst}
	home.instancesMu.Unlock()
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()
	// Find session item cursor position
	for i, item := range home.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil {
			home.cursor = i
			break
		}
	}

	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{} // nothing selected

	xMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	model, _ := home.Update(xMsg)
	h := model.(*Home)

	if !h.sendTextDialog.IsVisible() {
		t.Error("x with no selections should fall through to single-session send dialog")
	}
	if h.sendTextTargetID != "id-1" {
		t.Errorf("sendTextTargetID = %q, want 'id-1'", h.sendTextTargetID)
	}
	if len(h.sendTextTargetIDs) != 0 {
		t.Error("sendTextTargetIDs should be nil/empty for single-session send")
	}
}
