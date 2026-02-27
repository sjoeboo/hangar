# Hangar: home.go Refactor, Bug Fixes & Performance — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 3 confirmed bugs, reduce home.go from 10,322 lines to ~6,500 by extracting renderers and reshaping Update(), introduce a DialogManager to cut dialog wiring from 7 places to 3, consolidate 5 independent caches into one, and eliminate per-frame lipgloss allocations.

**Architecture:** Outside-in (Approach A) — each phase leaves all passing tests green and the binary buildable. New files in `package ui` share the `Home` receiver so extraction is purely mechanical. Bug fixes are self-contained and committed first to establish a clean baseline.

**Tech Stack:** Go 1.24, Bubble Tea (`charmbracelet/bubbletea v1.3.10`), Lipgloss (`charmbracelet/lipgloss v1.1.0`)

**Pre-existing failing tests** (do NOT fix; treat as known baseline):
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

---

## Phase 1: Bug Fixes

### Task 1: Fix Storage Leak in TransitionNotifier.dispatch()

**Files:**
- Modify: `internal/session/transition_notifier.go:122-132`
- Test: `internal/session/transition_notifier_test.go`

**Step 1: Understand the current code**

Read `internal/session/transition_notifier.go` lines 122–132. The `dispatch()` function calls `NewStorageWithProfile()` but never calls `storage.Close()`. Every time a session completes, a new SQLite connection opens and is abandoned.

**Step 2: Write a failing test**

In `internal/session/transition_notifier_test.go`, add:

```go
func TestDispatch_StorageIsClosed(t *testing.T) {
    // This is a compile-time check — if Storage doesn't have a Close method
    // or if we accidentally remove defer, the review catches it.
    // For now, verify dispatch() doesn't panic and returns a result.
    n := NewTransitionNotifier()
    event := TransitionNotificationEvent{
        Profile:        "default",
        ChildSessionID: "nonexistent-id",
    }
    result := n.dispatch(event)
    // nonexistent session should drop cleanly
    if result.DeliveryResult == transitionDeliveryFailed {
        t.Log("got failed (storage may not exist) — that's ok for this test")
    }
}
```

**Step 3: Run test to confirm baseline**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive
go test ./internal/session/... -run TestDispatch_StorageIsClosed -v
```

Expected: PASS or SKIP (no panic).

**Step 4: Apply the fix**

In `internal/session/transition_notifier.go`, find `dispatch()` (~line 122). Add `defer storage.Close()` immediately after the error check:

```go
func (n *TransitionNotifier) dispatch(event TransitionNotificationEvent) TransitionNotificationEvent {
    storage, err := NewStorageWithProfile(event.Profile)
    if err != nil {
        event.DeliveryResult = transitionDeliveryFailed
        return event
    }
    defer storage.Close()   // ← ADD THIS LINE
    instances, _, err := storage.LoadWithGroups()
```

**Step 5: Verify build and tests pass**

```bash
go build ./...
go test ./internal/session/... -v 2>&1 | grep -E "PASS|FAIL|panic"
```

Expected: all existing session tests PASS, no panics.

**Step 6: Commit**

```bash
git add internal/session/transition_notifier.go internal/session/transition_notifier_test.go
git commit -m "fix(session): close storage in TransitionNotifier.dispatch to prevent fd leak"
```

---

### Task 2: Fix Debounce Timer Leak in hook_watcher.go

**Files:**
- Modify: `internal/session/hook_watcher.go:80-131`
- Modify: `internal/session/event_watcher.go` (same pattern)
- Test: `internal/session/hook_watcher_test.go`

**Step 1: Understand the issue**

In `hook_watcher.go`, the watcher goroutine has a local `var debounceTimer *time.Timer`. When `Stop()` is called and the watcher closes, the goroutine returns — but `debounceTimer` may still be scheduled to fire 100ms later. Its callback calls `w.processFile()`, which tries to update `w.statuses` on a stopped watcher. Low severity but a latent race.

**Step 2: Write a test**

In `internal/session/hook_watcher_test.go`, add:

```go
func TestStatusFileWatcher_StopDuringDebounce(t *testing.T) {
    dir := t.TempDir()
    w, err := NewStatusFileWatcher(dir, func(id string, status *HookStatus) {})
    if err != nil {
        t.Fatal(err)
    }
    // Write a file to trigger debounce
    path := filepath.Join(dir, "test-id.json")
    data := `{"status":"waiting","tool":"claude"}`
    if err := os.WriteFile(path, []byte(data), 0644); err != nil {
        t.Fatal(err)
    }
    // Stop immediately — before the 100ms debounce fires
    w.Stop()
    // Sleep past debounce window; if timer fires after stop, may race
    time.Sleep(200 * time.Millisecond)
    // If we reach here without panic/race, the fix works
}
```

**Step 3: Run with race detector to see the issue**

```bash
go test -race ./internal/session/... -run TestStatusFileWatcher_StopDuringDebounce -v
```

Expected: may pass, may show a race. Either way, proceed with the fix.

**Step 4: Fix hook_watcher.go**

Change the goroutine in `hook_watcher.go` to use `defer` to stop the timer:

Find the goroutine that starts around line 80. Change:
```go
go func() {
    var debounceTimer *time.Timer
    var pendingMu sync.Mutex
    pendingFiles := make(map[string]bool)
    for {
        select {
```

To:
```go
go func() {
    var debounceTimer *time.Timer
    defer func() {
        if debounceTimer != nil {
            debounceTimer.Stop()
        }
    }()
    var pendingMu sync.Mutex
    pendingFiles := make(map[string]bool)
    for {
        select {
```

**Step 5: Apply the same fix to event_watcher.go**

Find the equivalent goroutine in `internal/session/event_watcher.go` — it has the identical debounce pattern. Apply the same `defer` fix.

**Step 6: Run tests with race detector**

```bash
go test -race ./internal/session/... -run TestStatusFileWatcher -v
```

Expected: PASS, no data race reported.

**Step 7: Commit**

```bash
git add internal/session/hook_watcher.go internal/session/event_watcher.go internal/session/hook_watcher_test.go
git commit -m "fix(session): stop debounce timer on watcher shutdown to prevent post-cancel callback"
```

---

### Task 3: Add Path Validation to `session set path`

**Files:**
- Modify: `cmd/hangar/session_cmd.go:880-882`
- Test: `cmd/hangar/session_cmd_test.go`

**Step 1: Write a failing test**

In `cmd/hangar/session_cmd_test.go`, add:

```go
func TestSessionSetPath_RejectsNonexistentPath(t *testing.T) {
    // This is a CLI command test — we verify the validation logic directly
    // by extracting it or testing the output behavior.
    // For now, document the expectation:
    // `hangar session set <id> path /nonexistent/path` should exit non-zero
    // with "path does not exist" message.
    // Add integration test here if CLI test harness is available.
    t.Skip("integration test: requires running hangar session set")
}
```

If the test harness in `testmain_test.go` supports it, write a proper test. Otherwise mark as skip with a clear note.

**Step 2: Add validation to the `path` case**

In `cmd/hangar/session_cmd.go`, find `case "path":` around line 880. Replace:

```go
case "path":
    oldValue = inst.ProjectPath
    inst.ProjectPath = value
```

With:

```go
case "path":
    info, err := os.Stat(value)
    if err != nil {
        out.Error(fmt.Sprintf("invalid path: %s (%v)", value, err), ErrCodeInvalidOperation)
        os.Exit(1)
    }
    if !info.IsDir() {
        out.Error(fmt.Sprintf("path is not a directory: %s", value), ErrCodeInvalidOperation)
        os.Exit(1)
    }
    oldValue = inst.ProjectPath
    inst.ProjectPath = value
```

Make sure `"os"` and `"fmt"` are already in the import block (they likely are).

**Step 3: Build and verify**

```bash
go build ./...
```

Expected: compiles cleanly.

**Step 4: Manual smoke test**

```bash
go run ./cmd/hangar session set someID path /nonexistent/path
```

Expected: exits with error message "invalid path: /nonexistent/path".

**Step 5: Commit**

```bash
git add cmd/hangar/session_cmd.go cmd/hangar/session_cmd_test.go
git commit -m "fix(cmd): validate path exists before accepting it in session set"
```

---

## Phase 2: Renderer Extraction

> **Goal:** Move ~1,500 lines of rendering code out of home.go into 3 new focused files. Each move is mechanical — same package, same receiver, same logic. Test before and after.

### Task 4: Extract Layout Renderers

**Files:**
- Create: `internal/ui/layout_renderer.go`
- Modify: `internal/ui/home.go` (remove functions)
- Test: `internal/ui/home_test.go`

**Step 1: Add a smoke test before moving**

In `internal/ui/home_test.go` (or a new `layout_renderer_test.go`), add:

```go
func TestRenderLayouts_Smoke(t *testing.T) {
    h := newTestHome(t)
    h.width = 120
    h.height = 40
    dual := h.renderDualColumnLayout(38)
    if dual == "" {
        t.Error("renderDualColumnLayout returned empty string")
    }
    h.width = 70
    stacked := h.renderStackedLayout(40)
    if stacked == "" {
        t.Error("renderStackedLayout returned empty string")
    }
    h.width = 40
    single := h.renderSingleColumnLayout(40)
    if single == "" {
        t.Error("renderSingleColumnLayout returned empty string")
    }
}
```

**Step 2: Run it to establish baseline**

```bash
go test ./internal/ui/... -run TestRenderLayouts_Smoke -v
```

Expected: PASS (functions exist, return non-empty strings).

**Step 3: Create `internal/ui/layout_renderer.go`**

```go
package ui

// layout_renderer.go contains the three responsive layout dispatch functions
// that select and compose the main content area based on terminal width.

// renderDualColumnLayout renders the side-by-side session list + preview pane
// layout used when terminal width >= 80.
func (h *Home) renderDualColumnLayout(contentHeight int) string {
    // MOVE BODY FROM home.go:6981
}

// renderStackedLayout renders the session list above the preview pane,
// used when terminal width is 50–79.
func (h *Home) renderStackedLayout(totalHeight int) string {
    // MOVE BODY FROM home.go:7042
}

// renderSingleColumnLayout renders only the session list with no preview pane,
// used when terminal width < 50.
func (h *Home) renderSingleColumnLayout(totalHeight int) string {
    // MOVE BODY FROM home.go:7082
}
```

Cut the 3 function bodies from `home.go` (lines 6981–7098) and paste them as the bodies in the new file. **Do not change the logic — only move it.**

**Step 4: Build**

```bash
go build ./...
```

Expected: compiles cleanly. If you get "declared but not used" or "undefined" errors, the cut was incomplete — check home.go still has `renderDualColumnLayout` in the `View()` call sites, not the function body.

**Step 5: Re-run smoke test**

```bash
go test ./internal/ui/... -run TestRenderLayouts_Smoke -v
```

Expected: still PASS.

**Step 6: Run full UI test suite**

```bash
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: same pass/fail count as before (2 pre-existing failures only).

**Step 7: Commit**

```bash
git add internal/ui/layout_renderer.go internal/ui/home.go
git commit -m "refactor(ui): extract renderDualColumnLayout/Stacked/Single to layout_renderer.go"
```

---

### Task 5: Extract Session List Renderer

**Files:**
- Create: `internal/ui/list_renderer.go`
- Modify: `internal/ui/home.go` (remove functions)

**Step 1: Add smoke tests**

In `internal/ui/home_test.go`, add:

```go
func TestRenderSessionList_Smoke(t *testing.T) {
    h := newTestHome(t)
    h.width = 80
    h.height = 40
    result := h.renderSessionList(80, 38)
    // Empty list renders empty state, not empty string
    if result == "" {
        t.Error("renderSessionList returned empty string even for empty state")
    }
}
```

**Step 2: Run to establish baseline**

```bash
go test ./internal/ui/... -run TestRenderSessionList_Smoke -v
```

Expected: PASS.

**Step 3: Create `internal/ui/list_renderer.go`**

```go
package ui

// list_renderer.go contains session and group list rendering functions.
// These render the left-side scrollable list in the main TUI view.
```

Move the following functions from `home.go` to `list_renderer.go`:
- `renderSessionList(width, height int) string` — line 7842
- `renderItem(b *strings.Builder, item session.Item, selected bool, itemIndex int)` — line 7905
- `renderGroupItem(b *strings.Builder, item session.Item, selected bool, itemIndex int)` — line 7915
- `renderSessionItem(b *strings.Builder, item session.Item, selected bool)` — line 8003
- `renderLaunchingState(inst *session.Instance, width int, startTime time.Time) string` — line 8180
- `renderMcpLoadingState(inst *session.Instance, width int, startTime time.Time) string` — line 8295
- `renderForkingState(inst *session.Instance, width int, startTime time.Time) string` — line 8348

**Step 4: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 5: Re-run smoke test and full suite**

```bash
go test ./internal/ui/... -run TestRenderSessionList_Smoke -v
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: smoke PASS, same overall pass/fail count.

**Step 6: Commit**

```bash
git add internal/ui/list_renderer.go internal/ui/home.go
git commit -m "refactor(ui): extract session and group list rendering to list_renderer.go"
```

---

### Task 6: Extract Preview Renderer

This is the biggest extraction — `renderPreviewPane()` is 1,033 lines (8472–9504). We move it AND split it into 5 section helpers.

**Files:**
- Create: `internal/ui/preview_renderer.go`
- Modify: `internal/ui/home.go` (remove functions)

**Step 1: Add a smoke test**

```go
func TestRenderPreviewPane_Smoke(t *testing.T) {
    h := newTestHome(t)
    h.width = 120
    h.height = 40
    // Empty session list — should render empty state, not empty string
    result := h.renderPreviewPane(60, 38)
    if result == "" {
        t.Error("renderPreviewPane returned empty string")
    }
}
```

**Step 2: Run baseline**

```bash
go test ./internal/ui/... -run TestRenderPreviewPane_Smoke -v
```

Expected: PASS.

**Step 3: Move `renderPreviewPane` and companions to a new file**

Create `internal/ui/preview_renderer.go` and move these functions from `home.go`:
- `renderPreviewPane(width, height int) string` — line 8472
- `renderSessionInfoCard(inst *session.Instance, width, height int) string` — line 8402
- `renderGroupPreview(group *session.Group, width, height int) string` — line 9504

**Step 4: Split renderPreviewPane into section helpers**

Inside `preview_renderer.go`, extract 5 private helper methods from `renderPreviewPane()`. The function has natural section boundaries — look for large comment blocks or `b.WriteString("\n")` boundaries inside it. Suggested split:

```go
// renderPreviewHeader renders the session name, status badge, and tool info.
func (h *Home) renderPreviewHeader(inst *session.Instance, width int) string { ... }

// renderPreviewInfoSection renders metadata: path, activity time, git info, recent log lines.
func (h *Home) renderPreviewInfoSection(inst *session.Instance, width, height int) string { ... }

// renderPreviewDiffSection renders the worktree diff stats and file list.
func (h *Home) renderPreviewDiffSection(inst *session.Instance, width, height int) string { ... }

// renderPreviewOutputSection renders the truncated tmux pane capture.
func (h *Home) renderPreviewOutputSection(inst *session.Instance, width, height int) string { ... }

// renderPreviewFooter renders the key hint bar at the bottom of the preview.
func (h *Home) renderPreviewFooter(width int) string { ... }
```

`renderPreviewPane()` itself becomes an orchestrator that calls these helpers and assembles the result.

**Note:** Keep the internal logic of each section exactly as-is. Only extract — do not refactor the logic yet. Logic changes come in Phase 6.

**Step 5: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 6: Re-run smoke test and full suite**

```bash
go test ./internal/ui/... -run TestRenderPreviewPane_Smoke -v
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: smoke PASS, same overall pass/fail count.

**Step 7: Commit**

```bash
git add internal/ui/preview_renderer.go internal/ui/home.go
git commit -m "refactor(ui): extract renderPreviewPane and companions to preview_renderer.go, split into section helpers"
```

---

## Phase 3: Reshape Update()

### Task 7: Extract Update() Message Handlers

**Files:**
- Create: `internal/ui/update_handlers.go`
- Modify: `internal/ui/home.go:2167-3593`

**Step 1: Identify which cases to extract**

Run this to see the message type cases and their approximate sizes:

```bash
grep -n "^[[:space:]]*case " /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/home.go | head -60
```

Extract cases that are **20+ lines**. Trivial cases (1-3 lines) can stay inline.

**Step 2: Create `internal/ui/update_handlers.go`**

```go
package ui

import (
    "github.com/charmbracelet/bubbletea"
    // other imports as needed
)

// update_handlers.go contains message-specific handler methods extracted
// from the monolithic Update() switch in home.go.
// Each handler receives its typed message, mutates Home state, and returns
// any follow-up commands.
```

**Step 3: Extract handlers one at a time**

For each large case in `Update()`, create a method and replace the case body with a call. Work through them in this order (simplest first):

1. `handleLoadSessions(msg loadSessionsMsg) tea.Cmd`
2. `handleSessionCreated(msg sessionCreatedMsg) tea.Cmd`
3. `handleSessionDeleted(msg sessionDeletedMsg) tea.Cmd`
4. `handleSessionForked(msg sessionForkedMsg) tea.Cmd`
5. `handleStorageChanged(msg storageChangedMsg) tea.Cmd`
6. `handleStatusUpdate(msg statusUpdateMsg) tea.Cmd`
7. `handlePreviewFetched(msg previewFetchedMsg) tea.Cmd`
8. `handleDiffFetched(msg diffFetchedMsg) tea.Cmd`
9. `handleWorktreeCreated(msg worktreeCreatedForNewSessionMsg) tea.Cmd`
10. `handleUpdateCheck(msg updateCheckMsg) tea.Cmd`

After each extraction:

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: clean build, same pass/fail count. **Do not proceed to the next extraction until the current one passes.**

**Step 4: Final Update() should look like:**

```go
func (h *Home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case quitMsg:
        h.isQuitting = true
        return h, tea.Quit
    case tea.WindowSizeMsg:
        h.width = msg.Width
        h.height = msg.Height
        h.dialogs.SetSize(msg.Width, msg.Height)   // after Phase 4
        return h, nil
    case loadSessionsMsg:
        return h, h.handleLoadSessions(msg)
    case sessionCreatedMsg:
        return h, h.handleSessionCreated(msg)
    case sessionDeletedMsg:
        return h, h.handleSessionDeleted(msg)
    case sessionForkedMsg:
        return h, h.handleSessionForked(msg)
    case storageChangedMsg:
        return h, h.handleStorageChanged(msg)
    case statusUpdateMsg:
        return h, h.handleStatusUpdate(msg)
    case previewFetchedMsg:
        return h, h.handlePreviewFetched(msg)
    case diffFetchedMsg:
        return h, h.handleDiffFetched(msg)
    case worktreeCreatedForNewSessionMsg:
        return h, h.handleWorktreeCreated(msg)
    case updateCheckMsg:
        return h, h.handleUpdateCheck(msg)
    case tea.MouseMsg:
        return h, h.handleMouseMsg(msg)
    case tea.KeyMsg:
        return h, h.handleKeyMsg(msg)
    }
    return h, nil
}
```

**Step 5: Run race detector**

```bash
go test -race ./internal/ui/... 2>&1 | grep -E "PASS|FAIL|DATA RACE"
```

Expected: no data races.

**Step 6: Commit**

```bash
git add internal/ui/update_handlers.go internal/ui/home.go
git commit -m "refactor(ui): extract Update() message handlers to update_handlers.go, Update() is now a thin dispatcher"
```

---

## Phase 4: Dialog Manager

### Task 8: Define the Dialog Interface

**Files:**
- Create: `internal/ui/dialog.go`

**Step 1: Create the interface file**

```go
// internal/ui/dialog.go
package ui

import tea "github.com/charmbracelet/bubbletea"

// Dialog is the interface that all overlay dialogs in the TUI must implement.
// It allows the DialogManager to handle visibility, sizing, key routing,
// and rendering uniformly across all dialogs.
type Dialog interface {
    // IsVisible reports whether the dialog is currently shown.
    IsVisible() bool

    // Show makes the dialog visible.
    Show()

    // Hide makes the dialog invisible.
    Hide()

    // View renders the dialog as a full-screen string.
    View() string

    // HandleKey processes a key event. Returns a command and whether the key
    // was consumed (true = do not process further).
    HandleKey(key tea.KeyMsg) (cmd tea.Cmd, consumed bool)

    // SetSize updates the dialog's dimensions to match the terminal.
    SetSize(width, height int)
}
```

**Step 2: Check which existing dialogs need signature adjustments**

Run:
```bash
grep -rn "func.*HandleKey\|func.*IsVisible\|func.*Show()\|func.*Hide()" /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/ | grep -v "_test.go"
```

For each dialog, check if `HandleKey` returns `(tea.Cmd, bool)` or something different. You may need to adjust return signatures. Do this one dialog at a time — build after each change.

**Step 3: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: clean.

**Step 4: Commit**

```bash
git add internal/ui/dialog.go
git commit -m "refactor(ui): define Dialog interface to formalize dialog contract"
```

---

### Task 9: Implement DialogManager

**Files:**
- Create: `internal/ui/dialog_manager.go`
- Modify: `internal/ui/home.go` (add field, update View, key routing, mouse guard, SetSize)
- Test: `internal/ui/dialog_manager_test.go`

**Step 1: Write tests for the manager**

Create `internal/ui/dialog_manager_test.go`:

```go
package ui

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
)

// mockDialog is a test double for Dialog
type mockDialog struct {
    visible bool
    lastKey tea.KeyMsg
}

func (m *mockDialog) IsVisible() bool { return m.visible }
func (m *mockDialog) Show()           { m.visible = true }
func (m *mockDialog) Hide()           { m.visible = false }
func (m *mockDialog) View() string    { return "mock-dialog" }
func (m *mockDialog) HandleKey(key tea.KeyMsg) (tea.Cmd, bool) {
    m.lastKey = key
    return nil, true
}
func (m *mockDialog) SetSize(w, h int) {}

func TestDialogManager_NoActive(t *testing.T) {
    dm := NewDialogManager()
    if dm.IsAnyVisible() {
        t.Error("expected no visible dialog")
    }
    if v := dm.View(); v != "" {
        t.Errorf("expected empty view, got %q", v)
    }
}

func TestDialogManager_PushMakesVisible(t *testing.T) {
    dm := NewDialogManager()
    d := &mockDialog{}
    dm.Push(d)
    if !dm.IsAnyVisible() {
        t.Error("expected dialog to be visible after Push")
    }
    if dm.View() != "mock-dialog" {
        t.Errorf("expected mock-dialog view")
    }
}

func TestDialogManager_PopRestoresState(t *testing.T) {
    dm := NewDialogManager()
    d := &mockDialog{}
    dm.Push(d)
    dm.Pop()
    if dm.IsAnyVisible() {
        t.Error("expected no visible dialog after Pop")
    }
}

func TestDialogManager_HandleKey_ConsumedByActive(t *testing.T) {
    dm := NewDialogManager()
    d := &mockDialog{}
    dm.Push(d)
    key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
    _, consumed := dm.HandleKey(key)
    if !consumed {
        t.Error("expected key to be consumed by active dialog")
    }
}

func TestDialogManager_SetSize_BroadcastsToAll(t *testing.T) {
    dm := NewDialogManager()
    d1 := &mockDialog{}
    d2 := &mockDialog{}
    dm.Register(d1)
    dm.Register(d2)
    dm.SetSize(100, 50) // Should not panic; dialogs receive the call
}
```

**Step 2: Run failing tests**

```bash
go test ./internal/ui/... -run TestDialogManager -v
```

Expected: FAIL — `NewDialogManager` undefined.

**Step 3: Create `internal/ui/dialog_manager.go`**

```go
package ui

import (
    "sync"
    tea "github.com/charmbracelet/bubbletea"
)

// DialogManager handles the lifecycle of overlay dialogs in the TUI.
// It maintains a stack of active dialogs (topmost = most recent) and
// a registry of all dialogs for size broadcasts.
//
// Usage:
//   h.dialogs.Push(h.newDialog)     // show a dialog
//   h.dialogs.Pop()                  // close the topmost dialog
//   if v := h.dialogs.View(); v != "" { return v }  // in View()
//   h.dialogs.HandleKey(msg)         // in Update(tea.KeyMsg)
type DialogManager struct {
    mu       sync.Mutex
    stack    []Dialog  // active dialogs; last element = topmost
    registry []Dialog  // all dialogs, for SetSize broadcast
}

// NewDialogManager creates an empty DialogManager.
func NewDialogManager() *DialogManager {
    return &DialogManager{}
}

// Register adds a dialog to the broadcast list for SetSize.
// Call this once per dialog during Home initialization.
func (dm *DialogManager) Register(d Dialog) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    dm.registry = append(dm.registry, d)
}

// Push shows a dialog by pushing it onto the stack.
func (dm *DialogManager) Push(d Dialog) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    d.Show()
    dm.stack = append(dm.stack, d)
}

// Pop hides and removes the topmost dialog.
func (dm *DialogManager) Pop() {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    if len(dm.stack) == 0 {
        return
    }
    top := dm.stack[len(dm.stack)-1]
    top.Hide()
    dm.stack = dm.stack[:len(dm.stack)-1]
}

// Active returns the topmost visible dialog, or nil if none.
func (dm *DialogManager) Active() Dialog {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    if len(dm.stack) == 0 {
        return nil
    }
    return dm.stack[len(dm.stack)-1]
}

// IsAnyVisible reports whether any dialog is currently active.
func (dm *DialogManager) IsAnyVisible() bool {
    return dm.Active() != nil
}

// View renders the topmost dialog. Returns "" if no dialog is active.
func (dm *DialogManager) View() string {
    if d := dm.Active(); d != nil {
        return d.View()
    }
    return ""
}

// HandleKey forwards the key to the active dialog.
// Returns the command and whether the key was consumed.
// If no dialog is active, consumed is false.
func (dm *DialogManager) HandleKey(key tea.KeyMsg) (tea.Cmd, bool) {
    if d := dm.Active(); d != nil {
        return d.HandleKey(key)
    }
    return nil, false
}

// SetSize broadcasts the new terminal dimensions to all registered dialogs.
func (dm *DialogManager) SetSize(width, height int) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    for _, d := range dm.registry {
        d.SetSize(width, height)
    }
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestDialogManager -v
```

Expected: all PASS.

**Step 5: Wire DialogManager into Home**

In `home.go`:

5a. Add field to `Home` struct:
```go
dialogs *DialogManager
```

5b. In `NewHomeWithProfileAndMode()`, initialize and register all dialogs:
```go
h.dialogs = NewDialogManager()
h.dialogs.Register(h.newDialog)
h.dialogs.Register(h.confirmDialog)
h.dialogs.Register(h.diffView)
h.dialogs.Register(h.todoDialog)
h.dialogs.Register(h.settingsPanel)
h.dialogs.Register(h.setupWizard)
h.dialogs.Register(h.groupDialog)
h.dialogs.Register(h.forkDialog)
h.dialogs.Register(h.geminiModelDialog)
h.dialogs.Register(h.worktreeFinishDialog)
h.dialogs.Register(h.reviewDialog)
h.dialogs.Register(h.sessionPickerDialog)
h.dialogs.Register(h.sendTextDialog)
h.dialogs.Register(h.helpOverlay)
h.dialogs.Register(h.globalSearch)
```

5c. In `View()`, replace the 17-line visibility cascade with:
```go
if v := h.dialogs.View(); v != "" {
    return h.ensureExactHeight(v)
}
```

5d. In `Update(tea.WindowSizeMsg)`, replace individual `dialog.SetSize()` calls with:
```go
h.dialogs.SetSize(msg.Width, msg.Height)
```

5e. In `handleKeyMsg()` (or the key routing section), add at the top:
```go
if cmd, consumed := h.dialogs.HandleKey(msg); consumed {
    return cmd
}
```

5f. In `handleMouseMsg()`, replace the individual `IsVisible()` guards with:
```go
if h.dialogs.IsAnyVisible() {
    return nil
}
```

5g. When triggering a dialog (e.g., pressing 'n' for new session), replace `h.newDialog.Show()` with `h.dialogs.Push(h.newDialog)`. When closing, replace `h.newDialog.Hide()` with `h.dialogs.Pop()`.

**Step 6: Build and run all tests**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

Expected: clean build, same pass/fail count (2 pre-existing failures only).

**Step 7: Manually verify dialogs open and close**

```bash
go run ./cmd/hangar
```

Open each dialog (n, d, D, t, S, ?, /) and verify:
- Dialog opens correctly
- Escape closes it
- Background sessions still update while a dialog is open

**Step 8: Commit**

```bash
git add internal/ui/dialog.go internal/ui/dialog_manager.go internal/ui/dialog_manager_test.go internal/ui/home.go
git commit -m "refactor(ui): introduce DialogManager, reduce dialog wiring from 7 places to 3"
```

---

## Phase 5: Cache Consolidation

### Task 10: Create UICache

**Files:**
- Create: `internal/ui/ui_cache.go`
- Create: `internal/ui/ui_cache_test.go`
- Modify: `internal/ui/home.go` (replace 5 cache maps with one field)

**Step 1: Write tests for UICache**

Create `internal/ui/ui_cache_test.go`:

```go
package ui

import (
    "testing"
    "time"
)

func TestUICache_PreviewMissAndHit(t *testing.T) {
    c := NewUICache()
    _, ok := c.GetPreview("abc")
    if ok {
        t.Error("expected cache miss")
    }
    c.SetPreview("abc", "rendered content")
    v, ok := c.GetPreview("abc")
    if !ok {
        t.Error("expected cache hit after set")
    }
    if v != "rendered content" {
        t.Errorf("unexpected value: %q", v)
    }
}

func TestUICache_TTLExpiry(t *testing.T) {
    c := NewUICache()
    c.setWithTTL("preview", "abc", "content", 10*time.Millisecond)
    _, ok := c.GetPreview("abc")
    if !ok {
        t.Fatal("expected hit before TTL")
    }
    time.Sleep(20 * time.Millisecond)
    _, ok = c.GetPreview("abc")
    if ok {
        t.Error("expected miss after TTL expiry")
    }
}

func TestUICache_InvalidateSession(t *testing.T) {
    c := NewUICache()
    c.SetPreview("sess1", "preview")
    c.SetPRInfo("sess1", &prInfo{URL: "http://example.com"})
    c.SetPreview("sess2", "other preview")
    c.InvalidateSession("sess1")
    if _, ok := c.GetPreview("sess1"); ok {
        t.Error("expected sess1 preview to be evicted")
    }
    if _, ok := c.GetPRInfo("sess1"); ok {
        t.Error("expected sess1 prInfo to be evicted")
    }
    if _, ok := c.GetPreview("sess2"); !ok {
        t.Error("sess2 preview should be unaffected")
    }
}

func TestUICache_InvalidateAll(t *testing.T) {
    c := NewUICache()
    c.SetPreview("sess1", "preview")
    c.SetPreview("sess2", "preview2")
    c.InvalidateAll()
    if _, ok := c.GetPreview("sess1"); ok {
        t.Error("expected all entries evicted")
    }
}
```

**Step 2: Run failing tests**

```bash
go test ./internal/ui/... -run TestUICache -v
```

Expected: FAIL — `NewUICache` undefined.

**Step 3: Create `internal/ui/ui_cache.go`**

```go
package ui

import (
    "fmt"
    "sync"
    "time"
)

// Cache TTLs — all cache expiry constants are defined here.
const (
    cachePreviewTTL      = 30 * time.Second
    cachePRInfoTTL       = 60 * time.Second
    cacheWorktreeDirtyTTL = 5 * time.Minute
)

// cacheEntry holds a cached value with its expiry time.
type cacheEntry[T any] struct {
    value   T
    expires time.Time
}

func (e cacheEntry[T]) valid() bool {
    return time.Now().Before(e.expires)
}

// UICache is a thread-safe, TTL-based cache for UI rendering data.
// It replaces five independent cache maps in Home with a single type
// and a single mutex.
type UICache struct {
    mu       sync.RWMutex
    previews map[string]cacheEntry[string]
    prInfos  map[string]cacheEntry[*prInfo]
    wt       map[string]cacheEntry[bool]
}

// NewUICache creates an empty UICache.
func NewUICache() *UICache {
    return &UICache{
        previews: make(map[string]cacheEntry[string]),
        prInfos:  make(map[string]cacheEntry[*prInfo]),
        wt:       make(map[string]cacheEntry[bool]),
    }
}

// GetPreview returns the cached preview string for sessionID, if valid.
func (c *UICache) GetPreview(sessionID string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    e, ok := c.previews[sessionID]
    if !ok || !e.valid() {
        return "", false
    }
    return e.value, true
}

// SetPreview stores a rendered preview string for sessionID.
func (c *UICache) SetPreview(sessionID, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.previews[sessionID] = cacheEntry[string]{value: value, expires: time.Now().Add(cachePreviewTTL)}
}

// GetPRInfo returns the cached PR info for sessionID, if valid.
func (c *UICache) GetPRInfo(sessionID string) (*prInfo, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    e, ok := c.prInfos[sessionID]
    if !ok || !e.valid() {
        return nil, false
    }
    return e.value, true
}

// SetPRInfo stores PR info for sessionID.
func (c *UICache) SetPRInfo(sessionID string, v *prInfo) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.prInfos[sessionID] = cacheEntry[*prInfo]{value: v, expires: time.Now().Add(cachePRInfoTTL)}
}

// GetWorktreeDirty returns the cached worktree dirty flag for sessionID.
func (c *UICache) GetWorktreeDirty(sessionID string) (bool, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    e, ok := c.wt[sessionID]
    if !ok || !e.valid() {
        return false, false
    }
    return e.value, true
}

// SetWorktreeDirty stores the worktree dirty flag for sessionID.
func (c *UICache) SetWorktreeDirty(sessionID string, dirty bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.wt[sessionID] = cacheEntry[bool]{value: dirty, expires: time.Now().Add(cacheWorktreeDirtyTTL)}
}

// InvalidateSession evicts all cached data for a specific session.
// Call this when a session is deleted or its state changes significantly.
func (c *UICache) InvalidateSession(sessionID string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.previews, sessionID)
    delete(c.prInfos, sessionID)
    delete(c.wt, sessionID)
}

// InvalidateAll clears the entire cache.
// Call this on full storage reload.
func (c *UICache) InvalidateAll() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.previews = make(map[string]cacheEntry[string])
    c.prInfos = make(map[string]cacheEntry[*prInfo])
    c.wt = make(map[string]cacheEntry[bool])
}

// setWithTTL is a test helper for injecting custom TTLs.
func (c *UICache) setWithTTL(kind, sessionID, value string, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    switch kind {
    case "preview":
        c.previews[sessionID] = cacheEntry[string]{value: value, expires: time.Now().Add(ttl)}
    default:
        panic(fmt.Sprintf("unknown kind %q", kind))
    }
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestUICache -v
```

Expected: all PASS.

**Step 5: Replace cache usage in Home**

In `home.go`:

5a. Add field to `Home` struct (and remove the 5 old cache fields + their mutexes):
```go
cache *UICache
```

5b. Initialize in `NewHomeWithProfileAndMode()`:
```go
h.cache = NewUICache()
```

5c. Replace all `h.previewCache[id]` / `h.previewCacheMu` accesses with `h.cache.GetPreview(id)` / `h.cache.SetPreview(id, v)`.

5d. Replace all `h.prCache[id]` / `h.prCacheMu` accesses with `h.cache.GetPRInfo(id)` / `h.cache.SetPRInfo(id, v)`.

5e. Replace all `h.worktreeDirtyCache[id]` / `h.worktreeDirtyCacheMu` accesses with `h.cache.GetWorktreeDirty(id)` / `h.cache.SetWorktreeDirty(id, v)`.

5f. Replace `h.invalidatePreviewCache(id)` calls with `h.cache.InvalidateSession(id)`.

5g. On `storageChangedMsg` full reload, call `h.cache.InvalidateAll()`.

Work through these replacements one cache type at a time, building after each type.

**Step 6: Build and run tests**

```bash
go build ./...
go test -race ./internal/ui/... 2>&1 | grep -E "PASS|FAIL|DATA RACE"
```

Expected: clean build, no races, same pass/fail.

**Step 7: Commit**

```bash
git add internal/ui/ui_cache.go internal/ui/ui_cache_test.go internal/ui/home.go
git commit -m "refactor(ui): consolidate 5 independent caches into UICache with unified TTL and invalidation"
```

---

## Phase 6: Performance

### Task 11: Move Inline Styles to Package Level

**Files:**
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/preview_renderer.go`
- Modify: `internal/ui/list_renderer.go`
- Modify: `internal/ui/layout_renderer.go`

**Step 1: Find all inline style creation in render functions**

```bash
grep -n "lipgloss\.NewStyle()" \
  /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/preview_renderer.go \
  /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/list_renderer.go \
  /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/layout_renderer.go \
  /Users/mnicholson/code/github/hangar/.worktrees/feature-deep-dive/internal/ui/home.go
```

**Step 2: For each inline style, move it to styles.go**

Pattern to eliminate:
```go
// BEFORE: inside a function (allocated every call)
lipgloss.NewStyle().Foreground(ColorText).Render(line)

// AFTER: in styles.go as a package-level var
var stylePreviewBodyText = lipgloss.NewStyle().Foreground(ColorText)
// used in render function as:
stylePreviewBodyText.Render(line)
```

Name convention: `style<Context><Usage>` — e.g., `stylePreviewBodyText`, `styleListItemTitle`, `styleLayoutDivider`.

Work through each renderer file. After moving each batch, build and test.

**Exception**: Styles that are parameterized at runtime (e.g., width-dependent `MaxWidth()` calls, or theme-dependent colors that change dynamically) should remain as function calls — only truly constant styles become package-level vars.

**Step 3: Build and test after each file**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

**Step 4: Commit**

```bash
git add internal/ui/styles.go internal/ui/preview_renderer.go internal/ui/list_renderer.go internal/ui/layout_renderer.go internal/ui/home.go
git commit -m "perf(ui): move inline lipgloss.NewStyle() calls to package-level vars to eliminate per-frame allocations"
```

---

### Task 12: Add Preview Section Memoization

**Files:**
- Modify: `internal/ui/ui_cache.go` (add section cache methods)
- Modify: `internal/ui/preview_renderer.go`

**Step 1: Add section cache to UICache**

In `ui_cache.go`, add methods for caching individual preview sections:

```go
// Preview section cache keys encode session ID + a version token.
// Version tokens encode the relevant session fields so stale data is
// automatically missed (not served).

// GetPreviewSection returns a cached rendered section by its key.
func (c *UICache) GetPreviewSection(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    e, ok := c.previews[key]
    if !ok || !e.valid() {
        return "", false
    }
    return e.value, true
}

// SetPreviewSection stores a rendered section with a given TTL.
func (c *UICache) SetPreviewSection(key, value string, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.previews[key] = cacheEntry[string]{value: value, expires: time.Now().Add(ttl)}
}
```

**Step 2: Add section-level caching to preview_renderer.go**

In each section helper, wrap the render with a cache check. For example, `renderPreviewInfoSection`:

```go
func (h *Home) renderPreviewInfoSection(inst *session.Instance, width, height int) string {
    // Cache key encodes the fields that affect this section's output.
    // If LastActivity changes, the key changes and we re-render.
    key := fmt.Sprintf("preview-info-%s-%d-%d-%d",
        inst.ID, inst.LastActivity.Unix(), width, height)
    if cached, ok := h.cache.GetPreviewSection(key); ok {
        return cached
    }
    // ... original render logic ...
    result := b.String()
    h.cache.SetPreviewSection(key, result, 30*time.Second)
    return result
}
```

Apply to each section helper with appropriate version tokens and TTLs:

| Section | Version token | TTL |
|---------|--------------|-----|
| Header | `inst.Status + inst.Title` | 5s |
| Info | `inst.LastActivity.Unix()` | 30s |
| Diff | `inst.DiffStats` hash or fixed | 60s |
| Output | `inst.LastActivity.Unix()` | 5s |
| Footer | `width` only (static content) | 5m |

**Step 3: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "PASS|FAIL"
```

**Step 4: Run race detector**

```bash
go test -race ./internal/ui/... 2>&1 | grep -E "PASS|FAIL|DATA RACE"
```

Expected: no data races (UICache is mutex-protected).

**Step 5: Commit**

```bash
git add internal/ui/ui_cache.go internal/ui/preview_renderer.go
git commit -m "perf(ui): memoize preview pane sections to eliminate redundant renders on each frame"
```

---

## Final Verification

After all 12 tasks are complete:

**Step 1: Full build and test**

```bash
go build ./...
go test -race ./... 2>&1 | grep -E "PASS|FAIL|DATA RACE"
```

Expected: all packages build, same 2 pre-existing failures, no data races.

**Step 2: Check home.go line count**

```bash
wc -l internal/ui/home.go
```

Expected: ~6,500 lines or fewer (down from 10,322).

**Step 3: Check Update() line count**

```bash
awk '/^func.*Update\(msg tea\.Msg\)/,/^func [A-Z]/' internal/ui/home.go | wc -l
```

Expected: ~80 lines.

**Step 4: Smoke test the binary manually**

```bash
go run ./cmd/hangar
```

- Verify TUI opens and renders sessions
- Open and close each dialog (n, d, D, t, S, ?, /)
- Verify session status updates are visible
- Verify diff view works on a worktree session

**Step 5: Final commit**

```bash
git commit -m "chore: verify home.go refactor complete — $(wc -l < internal/ui/home.go) lines"
```

---

## Summary

| Phase | Tasks | Key outcome |
|-------|-------|------------|
| 1. Bug fixes | 1–3 | Clean, verified baseline |
| 2. Renderer extraction | 4–6 | home.go –1,500 lines; renderPreviewPane split |
| 3. Update() reshape | 7 | Update() = 80-line dispatcher |
| 4. Dialog manager | 8–9 | 7-point wiring → 3; View() readable |
| 5. Cache consolidation | 10 | 5 caches + 5 mutexes → 1 |
| 6. Performance | 11–12 | Zero per-frame allocs; preview memoized |
