# Bulk Session Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add vim-style visual mode (`V`) that lets users select multiple sessions and apply bulk delete (with worktree cleanup), send message, or restart.

**Architecture:** Two new fields on `Home` (`bulkSelectMode bool`, `selectedSessionIDs map[string]bool`) drive a mode-aware rendering path and action dispatch. The existing `selectionPrefix` slot in `renderSessionItem` shows `□`/`☑` checkboxes. Action keys (`d`, `x`, `R`) check the mode and dispatch to new bulk variants; falling through to single-session behavior when nothing is selected.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), lipgloss, existing Hangar `internal/ui` patterns.

---

### Known pre-existing failing tests (do not fix, do not break further)
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

Run tests with: `go test ./internal/ui/... -v`

---

## Task 1: Add bulk-select state and V key toggle

**Files:**
- Modify: `internal/ui/home.go:315-318` (after `preview *Preview`, before closing `}` of Home struct)
- Modify: `internal/ui/home.go` (NewHome initializer — search for `undoStack:`)
- Modify: `internal/ui/home.go:4352-4356` (before `case "d":` — add `case "V":` and Esc guard)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing tests**

Add to `internal/ui/home_test.go`:

```go
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
```

**Step 2: Run to verify they fail**

```
go test ./internal/ui/... -run "TestBulkSelectMode" -v
```
Expected: FAIL with `home.bulkSelectMode undefined`

**Step 3: Add fields to Home struct**

In `internal/ui/home.go`, after the `preview *Preview` line (~line 317), add:

```go
	// Bulk select mode
	bulkSelectMode    bool
	selectedSessionIDs map[string]bool
```

**Step 4: Initialize selectedSessionIDs in NewHome**

Find the `NewHome()` function (search for `undoStack:` to locate the struct literal). Add:

```go
		selectedSessionIDs: make(map[string]bool),
```

**Step 5: Add V key handler**

In `home.go`, find the main key switch (around `case "d":` at ~line 4356). Add before it:

```go
	case "V":
		if h.bulkSelectMode {
			h.bulkSelectMode = false
			h.selectedSessionIDs = make(map[string]bool)
		} else {
			h.bulkSelectMode = true
		}
		return h, nil
```

**Step 6: Add Esc guard for bulk mode**

Find the `case "esc":` handler in the main key switch (not inside a dialog handler). It likely checks various dialogs. Add at the very top of that case, before other checks:

```go
	case "esc":
		if h.bulkSelectMode {
			h.bulkSelectMode = false
			h.selectedSessionIDs = make(map[string]bool)
			return h, nil
		}
		// ... existing esc handling continues
```

**Step 7: Run tests to verify they pass**

```
go test ./internal/ui/... -run "TestBulkSelectMode" -v
```
Expected: PASS

**Step 8: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): add bulk-select mode state and V key toggle"
```

---

## Task 2: Space key toggles session selection in bulk mode

**Files:**
- Modify: `internal/ui/home.go` (add `case " ":` in main key switch)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

Add to `internal/ui/home_test.go`:

```go
func TestBulkSelectMode_SpaceTogglesSelection(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	// Set up one session at cursor
	inst := &session.Instance{ID: "test-1", Title: "test-session"}
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"test-1": inst}
	home.rebuildFlatItems()
	home.cursor = 0

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

	// Manually place a group item at cursor
	home.flatItems = []session.Item{
		{Type: session.ItemTypeGroup, Path: "default", Group: &session.Group{Name: "default"}},
	}
	home.cursor = 0

	// Enter bulk mode and press space
	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
	model, _ := home.Update(vMsg)
	h := model.(*Home)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ = h.Update(spaceMsg)
	h = model.(*Home)
	if len(h.selectedSessionIDs) != 0 {
		t.Error("space on group should be a no-op")
	}
}
```

**Step 2: Run to verify they fail**

```
go test ./internal/ui/... -run "TestBulkSelectMode_Space" -v
```
Expected: FAIL

**Step 3: Add space handler**

In `home.go` main key switch, add after the `case "V":` block:

```go
	case " ":
		if h.bulkSelectMode && h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				id := item.Session.ID
				if h.selectedSessionIDs[id] {
					delete(h.selectedSessionIDs, id)
				} else {
					h.selectedSessionIDs[id] = true
				}
			}
		}
		return h, nil
```

**Step 4: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_Space" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): space key toggles session selection in bulk mode"
```

---

## Task 3: Render checkboxes in session rows

**Files:**
- Modify: `internal/ui/home.go:7633-7648` (`renderSessionItem` — the `selectionPrefix` block)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

```go
func TestBulkSelectMode_CheckboxRendering(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst := &session.Instance{ID: "test-1", Title: "my-session"}
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"test-1": inst}
	home.rebuildFlatItems()
	home.cursor = 0

	// In bulk mode, view should contain checkbox characters
	home.bulkSelectMode = true

	view := home.View()
	if !strings.Contains(view, "□") {
		t.Error("bulk mode should render □ for unselected session")
	}

	// After selecting, should show checked box
	home.selectedSessionIDs["test-1"] = true
	view = home.View()
	if !strings.Contains(view, "☑") {
		t.Error("selected session should render ☑")
	}
}
```

**Step 2: Run to verify it fails**

```
go test ./internal/ui/... -run "TestBulkSelectMode_CheckboxRendering" -v
```
Expected: FAIL (no checkbox chars in view)

**Step 3: Modify renderSessionItem**

Find the `selectionPrefix` block at ~line 7633:

```go
	// Selection indicator
	selectionPrefix := " "
	if selected {
		selectionPrefix = SessionSelectionPrefix.Render("▶")
		// ... styling
	}
```

Replace with:

```go
	// Selection indicator
	var selectionPrefix string
	if h.bulkSelectMode {
		if h.selectedSessionIDs[inst.ID] {
			selectionPrefix = SessionSelectionPrefix.Render("☑")
		} else {
			selectionPrefix = lipgloss.NewStyle().Foreground(ColorTextDim).Render("□")
		}
		if selected {
			// Apply selection styling for the focused row in bulk mode
			titleStyle = SessionTitleSelStyle
			toolStyle = SessionStatusSelStyle
			statusStyle = SessionStatusSelStyle
			status = statusStyle.Render(statusIcon)
			treeStyle = TreeConnectorSelStyle
			if item.IsSubSession && !item.ParentIsLastInGroup {
				groupIndent := strings.Repeat(treeEmpty, max(0, item.Level-2))
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		}
	} else {
		selectionPrefix = " "
		if selected {
			selectionPrefix = SessionSelectionPrefix.Render("▶")
			titleStyle = SessionTitleSelStyle
			toolStyle = SessionStatusSelStyle
			statusStyle = SessionStatusSelStyle
			status = statusStyle.Render(statusIcon)
			treeStyle = TreeConnectorSelStyle
			if item.IsSubSession && !item.ParentIsLastInGroup {
				groupIndent := strings.Repeat(treeEmpty, max(0, item.Level-2))
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		}
	}
```

**Step 4: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_Checkbox" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): render checkbox indicators in session rows during bulk mode"
```

---

## Task 4: Bulk-mode help bar

**Files:**
- Modify: `internal/ui/home.go:6839` (`renderHelpBar`)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

```go
func TestBulkSelectMode_HelpBarShown(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30
	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"a": true, "b": true}

	view := home.View()
	if !strings.Contains(view, "VISUAL") {
		t.Error("bulk mode should show VISUAL in help bar")
	}
	if !strings.Contains(view, "2 selected") {
		t.Error("bulk mode should show selection count")
	}
}
```

**Step 2: Run to verify it fails**

```
go test ./internal/ui/... -run "TestBulkSelectMode_HelpBar" -v
```

**Step 3: Add bulk-mode branch to renderHelpBar**

At the top of `renderHelpBar()`, before the `switch` statement, add:

```go
func (h *Home) renderHelpBar() string {
	if h.bulkSelectMode {
		return h.renderHelpBarBulkMode()
	}
	// Route to appropriate tier based on width
	switch {
	// ... existing code
```

Then add the new method:

```go
// renderHelpBarBulkMode renders the bulk-select mode hint bar
func (h *Home) renderHelpBarBulkMode() string {
	borderStyle := lipgloss.NewStyle().Foreground(ColorAccent)
	border := borderStyle.Render(strings.Repeat("─", max(0, h.width)))

	keyStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	labelStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

	count := len(h.selectedSessionIDs)
	countStr := fmt.Sprintf("%d selected", count)

	sep := dimStyle.Render("  ·  ")
	hint := labelStyle.Render("VISUAL") + "  " + dimStyle.Render(countStr) +
		sep + keyStyle.Render("spc") + dimStyle.Render(":toggle") +
		sep + keyStyle.Render("d") + dimStyle.Render(":delete") +
		sep + keyStyle.Render("x") + dimStyle.Render(":message") +
		sep + keyStyle.Render("R") + dimStyle.Render(":restart") +
		sep + keyStyle.Render("Esc") + dimStyle.Render(":cancel")

	raw := lipgloss.JoinVertical(lipgloss.Left, border, hint)
	return lipgloss.NewStyle().MaxWidth(h.width).Render(raw)
}
```

**Step 4: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_HelpBar" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): show bulk-mode hint bar with selection count and action keys"
```

---

## Task 5: ConfirmBulkDeleteSessions dialog type

**Files:**
- Modify: `internal/ui/confirm_dialog.go`
- Test: `internal/ui/confirm_dialog_test.go` (create if not exists)

**Step 1: Write failing test**

Create `internal/ui/confirm_dialog_test.go`:

```go
package ui

import (
	"strings"
	"testing"
)

func TestConfirmDialog_BulkDelete(t *testing.T) {
	d := NewConfirmDialog()
	d.SetSize(80, 24)

	d.ShowBulkDeleteSessions(
		[]string{"id-1", "id-2", "id-3"},
		[]string{"session-alpha", "session-beta [worktree]", "session-gamma"},
	)

	if !d.IsVisible() {
		t.Fatal("dialog should be visible")
	}
	if d.GetConfirmType() != ConfirmBulkDeleteSessions {
		t.Errorf("confirm type = %v, want ConfirmBulkDeleteSessions", d.GetConfirmType())
	}
	ids := d.GetTargetIDs()
	if len(ids) != 3 {
		t.Errorf("GetTargetIDs len = %d, want 3", len(ids))
	}

	view := d.View()
	if !strings.Contains(view, "3 sessions") {
		t.Error("view should mention 3 sessions")
	}
	if !strings.Contains(view, "session-alpha") {
		t.Error("view should list session names")
	}
	if !strings.Contains(view, "session-beta [worktree]") {
		t.Error("view should show worktree annotation")
	}
}
```

**Step 2: Run to verify it fails**

```
go test ./internal/ui/... -run "TestConfirmDialog_BulkDelete" -v
```

**Step 3: Add ConfirmBulkDeleteSessions to confirm_dialog.go**

In the `const` iota block at line 15, add:

```go
const (
	ConfirmDeleteSession ConfirmType = iota
	ConfirmDeleteGroup
	ConfirmQuitWithPool
	ConfirmCreateDirectory
	ConfirmInstallHooks
	ConfirmBulkDeleteSessions
	ConfirmBulkRestart
)
```

Add fields to `ConfirmDialog` struct after `mcpCount int`:

```go
	// Bulk operation fields
	targetIDs   []string // Multiple session IDs (for bulk operations)
	targetNames []string // Display names for bulk targets
```

Add `ShowBulkDeleteSessions` method:

```go
// ShowBulkDeleteSessions shows confirmation for bulk session deletion
func (c *ConfirmDialog) ShowBulkDeleteSessions(ids []string, names []string) {
	c.visible = true
	c.confirmType = ConfirmBulkDeleteSessions
	c.targetIDs = ids
	c.targetNames = names
	c.targetID = ""
	c.targetName = fmt.Sprintf("%d sessions", len(ids))
}

// ShowBulkRestart shows confirmation for bulk session restart
func (c *ConfirmDialog) ShowBulkRestart(count int) {
	c.visible = true
	c.confirmType = ConfirmBulkRestart
	c.targetID = ""
	c.targetName = fmt.Sprintf("%d sessions", count)
}

// GetTargetIDs returns the list of session IDs for bulk operations
func (c *ConfirmDialog) GetTargetIDs() []string {
	return c.targetIDs
}
```

Add the `ConfirmBulkDeleteSessions` case to `View()` switch, before the closing `}` of the switch:

```go
	case ConfirmBulkDeleteSessions:
		title = fmt.Sprintf("⚠️  Delete %s?", c.targetName)
		// Build session list (max 8 shown, then "and N more...")
		lines := c.targetNames
		shown := lines
		extra := 0
		if len(lines) > 8 {
			shown = lines[:8]
			extra = len(lines) - 8
		}
		listStr := ""
		for _, name := range shown {
			listStr += fmt.Sprintf("  • %s\n", name)
		}
		if extra > 0 {
			listStr += fmt.Sprintf("  … and %d more", extra)
		}
		warning = "This will PERMANENTLY KILL these tmux sessions:\n\n" + strings.TrimRight(listStr, "\n")
		details = "• Worktrees will be removed where applicable\n• Press Ctrl+Z after deletion to undo (one at a time)"
		borderColor = ColorRed

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorRed).Padding(0, 2).Bold(true).
			Render(fmt.Sprintf("y Delete %s", c.targetName))
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorAccent).Padding(0, 2).Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().Foreground(ColorTextDim).Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmBulkRestart:
		title = fmt.Sprintf("Restart %s?", c.targetName)
		warning = fmt.Sprintf("Restart all %s?", c.targetName)
		details = "Each session will be recreated and resumed."
		borderColor = ColorAccent

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorAccent).Padding(0, 2).Bold(true).
			Render("y Restart")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorRed).Padding(0, 2).Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().Foreground(ColorTextDim).Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)
```

Also update `Hide()` to clear bulk fields:

```go
func (c *ConfirmDialog) Hide() {
	c.visible = false
	c.targetID = ""
	c.targetName = ""
	c.targetIDs = nil
	c.targetNames = nil
}
```

**Step 4: Run tests**

```
go test ./internal/ui/... -run "TestConfirmDialog_BulkDelete" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/confirm_dialog.go internal/ui/confirm_dialog_test.go
git commit -m "feat(bulk): add ConfirmBulkDeleteSessions and ConfirmBulkRestart dialog types"
```

---

## Task 6: Bulk delete — d key dispatch and execution

**Files:**
- Modify: `internal/ui/home.go:4356-4386` (`case "d":`)
- Modify: `internal/ui/home.go:4774-4806` (`handleConfirmDialogKey`)
- Modify: `internal/ui/home.go` (add `bulkDeleteSessions` command + result message type + Update handler)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

```go
func TestBulkSelectMode_DKeyShowsBulkConfirm(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst1 := &session.Instance{ID: "id-1", Title: "sess-1"}
	inst2 := &session.Instance{ID: "id-2", Title: "sess-2"}
	home.instances = []*session.Instance{inst1, inst2}
	home.instanceByID = map[string]*session.Instance{"id-1": inst1, "id-2": inst2}
	home.rebuildFlatItems()

	// Enter bulk mode, select both sessions
	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"id-1": true, "id-2": true}

	// Press d
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

	inst := &session.Instance{ID: "id-1", Title: "sess-1"}
	home.instances = []*session.Instance{inst}
	home.instanceByID = map[string]*session.Instance{"id-1": inst}
	home.rebuildFlatItems()
	home.cursor = 0

	// Bulk mode but nothing selected
	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{}

	// Press d - should fall through to single-session confirm
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, _ := home.Update(dMsg)
	h := model.(*Home)

	if !h.confirmDialog.IsVisible() {
		t.Error("d with no selections should fall through to single-session delete confirm")
	}
	if h.confirmDialog.GetConfirmType() != ConfirmDeleteSession {
		t.Errorf("confirm type = %v, want ConfirmDeleteSession", h.confirmDialog.GetConfirmType())
	}
}
```

**Step 2: Run to verify they fail**

```
go test ./internal/ui/... -run "TestBulkSelectMode_DKey" -v
```

**Step 3: Modify case "d" in home.go**

Replace the current `case "d":` block at ~line 4356 with:

```go
	case "d":
		// Bulk mode: if any sessions are selected, show bulk confirm
		if h.bulkSelectMode && len(h.selectedSessionIDs) > 0 {
			var ids, names []string
			for _, item := range h.flatItems {
				if item.Type == session.ItemTypeSession && item.Session != nil {
					if h.selectedSessionIDs[item.Session.ID] {
						name := item.Session.Title
						if item.Session.IsWorktree() {
							name += " [worktree]"
						}
						ids = append(ids, item.Session.ID)
						names = append(names, name)
					}
				}
			}
			h.confirmDialog.ShowBulkDeleteSessions(ids, names)
			return h, nil
		}
		// Single-session delete (existing logic unchanged)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				inst := item.Session
				if inst.IsWorktree() {
					h.worktreeFinishDialog.SetSize(h.width, h.height)
					h.worktreeFinishDialog.Show(inst.ID, inst.Title, inst.WorktreeBranch, inst.WorktreeRepoRoot, inst.WorktreePath)
					h.prCacheMu.Lock()
					cachedPR, hasPRCached := h.prCache[inst.ID]
					h.prCacheMu.Unlock()
					if hasPRCached {
						h.worktreeFinishDialog.SetPR(cachedPR, true)
					}
					sid := inst.ID
					wtPath := inst.WorktreePath
					return h, func() tea.Msg {
						dirty, err := git.HasUncommittedChanges(wtPath)
						return worktreeDirtyCheckMsg{sessionID: sid, isDirty: dirty, err: err}
					}
				}
				h.confirmDialog.ShowDeleteSession(inst.ID, inst.Title)
			} else if item.Type == session.ItemTypeGroup && item.Path != session.DefaultGroupPath {
				h.confirmDialog.ShowDeleteGroup(item.Path, item.Group.Name)
			}
		}
		return h, nil
```

**Step 4: Add bulkDeletedMsg and bulkDeleteSessions**

After `sessionDeletedMsg` type (near ~line 5520), add:

```go
// bulkDeletedMsg signals that a bulk delete completed
type bulkDeletedMsg struct {
	deletedIDs []string
	killErrs   []error
}

// bulkDeleteSessions deletes multiple sessions, cleaning up worktrees as needed
func (h *Home) bulkDeleteSessions(insts []*session.Instance) tea.Cmd {
	type entry struct {
		id               string
		inst             *session.Instance
		isWorktree       bool
		worktreePath     string
		worktreeRepoRoot string
	}
	entries := make([]entry, len(insts))
	for i, inst := range insts {
		entries[i] = entry{
			id:               inst.ID,
			inst:             inst,
			isWorktree:       inst.IsWorktree(),
			worktreePath:     inst.WorktreePath,
			worktreeRepoRoot: inst.WorktreeRepoRoot,
		}
	}
	return func() tea.Msg {
		var deletedIDs []string
		var killErrs []error
		for _, e := range entries {
			err := e.inst.Kill()
			if e.isWorktree {
				_ = git.RemoveWorktree(e.worktreeRepoRoot, e.worktreePath, false)
				_ = git.PruneWorktrees(e.worktreeRepoRoot)
			}
			deletedIDs = append(deletedIDs, e.id)
			if err != nil {
				killErrs = append(killErrs, err)
			}
		}
		return bulkDeletedMsg{deletedIDs: deletedIDs, killErrs: killErrs}
	}
}
```

**Step 5: Handle ConfirmBulkDeleteSessions in handleConfirmDialogKey**

In `handleConfirmDialogKey` at ~line 4779, inside the `case "y", "Y":` → `switch h.confirmDialog.GetConfirmType()`:

```go
		case ConfirmBulkDeleteSessions:
			ids := h.confirmDialog.GetTargetIDs()
			var insts []*session.Instance
			for _, id := range ids {
				if inst := h.getInstanceByID(id); inst != nil {
					insts = append(insts, inst)
				}
			}
			h.confirmDialog.Hide()
			if len(insts) > 0 {
				return h, h.bulkDeleteSessions(insts)
			}
			return h, nil
```

Also in `case "n", "N", "esc":` the existing `h.confirmDialog.Hide()` already handles cancelling.

**Step 6: Handle bulkDeletedMsg in Update**

Find where `sessionDeletedMsg` is handled (~line 2480) and add after it:

```go
		case bulkDeletedMsg:
			for _, id := range msg.deletedIDs {
				if inst := h.getInstanceByID(id); inst != nil {
					h.undoStack = append(h.undoStack, deletedSessionEntry{
						instance:  inst,
						deletedAt: time.Now(),
					})
					if len(h.undoStack) > 10 {
						h.undoStack = h.undoStack[len(h.undoStack)-10:]
					}
					h.instancesMu.Lock()
					h.instances = removeInstance(h.instances, id)
					delete(h.instanceByID, id)
					h.instancesMu.Unlock()
				}
			}
			h.rebuildFlatItems()
			h.saveInstances()
			h.bulkSelectMode = false
			h.selectedSessionIDs = make(map[string]bool)
			if len(msg.killErrs) > 0 {
				h.setError(fmt.Errorf("deleted %d sessions (some kill errors)", len(msg.deletedIDs)))
			} else {
				h.setError(fmt.Errorf("deleted %d sessions. Ctrl+Z to undo (one at a time)", len(msg.deletedIDs)))
			}
			return h, nil
```

Note: find `removeInstance` helper or inline equivalent logic — search for existing usage of `removeInstance` to find the correct helper name in this codebase.

**Step 7: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_DKey" -v
```
Expected: PASS

**Step 8: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): bulk delete sessions with worktree cleanup via d key in visual mode"
```

---

## Task 7: Bulk send message — x key in bulk mode

**Files:**
- Modify: `internal/ui/home.go` — add `sendTextTargetIDs []string` field, modify `case "x":`, modify `handleSendTextDialogKey`
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

```go
func TestBulkSelectMode_XKeyOpensSendDialog(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst1 := &session.Instance{ID: "id-1", Title: "sess-1"}
	inst2 := &session.Instance{ID: "id-2", Title: "sess-2"}
	home.instances = []*session.Instance{inst1, inst2}
	home.instanceByID = map[string]*session.Instance{"id-1": inst1, "id-2": inst2}
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
}
```

**Step 2: Run to verify it fails**

```
go test ./internal/ui/... -run "TestBulkSelectMode_XKey" -v
```

**Step 3: Add sendTextTargetIDs field**

In the `Home` struct, near `sendTextTargetID string` (~line 149), add:

```go
	sendTextTargetIDs    []string // Multiple session IDs for bulk send
```

**Step 4: Modify case "x"**

Replace current `case "x":` (~line 4486):

```go
	case "x":
		// Bulk mode: send to all selected sessions
		if h.bulkSelectMode && len(h.selectedSessionIDs) > 0 {
			var ids []string
			for _, item := range h.flatItems {
				if item.Type == session.ItemTypeSession && item.Session != nil {
					if h.selectedSessionIDs[item.Session.ID] {
						ids = append(ids, item.Session.ID)
					}
				}
			}
			h.sendTextTargetIDs = ids
			h.sendTextTargetID = ""
			h.sendTextDialog.SetSize(h.width, h.height)
			h.sendTextDialog.Show(fmt.Sprintf("%d sessions", len(ids)))
			return h, nil
		}
		// Single-session send (existing logic)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				h.sendTextTargetID = item.Session.ID
				h.sendTextTargetIDs = nil
				h.sendTextDialog.SetSize(h.width, h.height)
				h.sendTextDialog.Show(item.Session.Title)
			}
		}
		return h, nil
```

**Step 5: Modify handleSendTextDialogKey for bulk**

Find `handleSendTextDialogKey` at ~line 9330. In the `SendTextActionConfirm` case, modify the send logic:

```go
	case SendTextActionConfirm:
		text := h.sendTextDialog.GetText()
		h.sendTextDialog.Hide()
		if text == "" {
			h.sendTextTargetID = ""
			h.sendTextTargetIDs = nil
			return h, nil
		}
		// Bulk send
		if len(h.sendTextTargetIDs) > 0 {
			ids := h.sendTextTargetIDs
			h.sendTextTargetIDs = nil
			h.bulkSelectMode = false
			h.selectedSessionIDs = make(map[string]bool)
			return h, func() tea.Msg {
				var lastErr error
				count := 0
				for _, id := range ids {
					h.instancesMu.RLock()
					inst := h.instanceByID[id]
					h.instancesMu.RUnlock()
					if inst == nil {
						continue
					}
					tmuxSess := inst.GetTmuxSession()
					if tmuxSess == nil {
						continue
					}
					if err := tmuxSess.SendKeys(text); err != nil {
						lastErr = err
					} else {
						count++
					}
				}
				if lastErr != nil {
					return sendTextResultMsg{targetTitle: fmt.Sprintf("%d sessions", len(ids)), err: lastErr}
				}
				return sendTextResultMsg{targetTitle: fmt.Sprintf("%d sessions", count)}
			}
		}
		// Single target (existing logic)
		if h.sendTextTargetID != "" {
			// ... existing single-send code unchanged
		}
```

Note: examine the existing `SendTextActionConfirm` logic carefully and graft in the bulk branch at the top without removing any single-session behavior.

**Step 6: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_XKey" -v
```
Expected: PASS

**Step 7: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): bulk send message to selected sessions via x key in visual mode"
```

---

## Task 8: Bulk restart — R key in bulk mode

**Files:**
- Modify: `internal/ui/home.go:4442-4459` (`case "R":`)
- Modify: `internal/ui/home.go` (`handleConfirmDialogKey`, new `bulkRestartSessions` cmd, result msg)
- Test: `internal/ui/home_test.go`

**Step 1: Write failing test**

```go
func TestBulkSelectMode_RKeyShowsBulkRestartConfirm(t *testing.T) {
	home := NewHome()
	home.width = 100
	home.height = 30

	inst1 := &session.Instance{ID: "id-1", Title: "sess-1"}
	inst2 := &session.Instance{ID: "id-2", Title: "sess-2"}
	home.instances = []*session.Instance{inst1, inst2}
	home.instanceByID = map[string]*session.Instance{"id-1": inst1, "id-2": inst2}
	home.rebuildFlatItems()

	home.bulkSelectMode = true
	home.selectedSessionIDs = map[string]bool{"id-1": true, "id-2": true}

	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
	model, _ := home.Update(rMsg)
	h := model.(*Home)

	if !h.confirmDialog.IsVisible() {
		t.Error("R in bulk mode should show confirm dialog")
	}
	if h.confirmDialog.GetConfirmType() != ConfirmBulkRestart {
		t.Errorf("confirm type = %v, want ConfirmBulkRestart", h.confirmDialog.GetConfirmType())
	}
}
```

**Step 2: Run to verify it fails**

```
go test ./internal/ui/... -run "TestBulkSelectMode_RKey" -v
```

**Step 3: Modify case "R"**

Replace current `case "R":` block at ~line 4442:

```go
	case "R":
		// Bulk mode: confirm restart of all selected sessions
		if h.bulkSelectMode && len(h.selectedSessionIDs) > 0 {
			h.confirmDialog.ShowBulkRestart(len(h.selectedSessionIDs))
			return h, nil
		}
		// Single-session restart (existing logic unchanged)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				if h.hasActiveAnimation(item.Session.ID) {
					h.setError(fmt.Errorf("session is starting, please wait..."))
					return h, nil
				}
				if item.Session.CanRestart() {
					h.resumingSessions[item.Session.ID] = time.Now()
					return h, h.restartSession(item.Session)
				}
			}
		}
		return h, nil
```

**Step 4: Add bulkRestartedMsg and bulkRestartSessions**

After `sessionRestartedMsg` (~line 5543), add:

```go
// bulkRestartedMsg signals that a bulk restart completed
type bulkRestartedMsg struct {
	restartedIDs []string
	errs         []error
}

// bulkRestartSessions restarts multiple sessions
func (h *Home) bulkRestartSessions(insts []*session.Instance) tea.Cmd {
	ids := make([]string, len(insts))
	for i, inst := range insts {
		ids[i] = inst.ID
	}
	return func() tea.Msg {
		var restartedIDs []string
		var errs []error
		for _, id := range ids {
			h.instancesMu.RLock()
			inst := h.instanceByID[id]
			h.instancesMu.RUnlock()
			if inst == nil {
				continue
			}
			if err := inst.Restart(); err != nil {
				errs = append(errs, err)
			} else {
				restartedIDs = append(restartedIDs, id)
			}
		}
		return bulkRestartedMsg{restartedIDs: restartedIDs, errs: errs}
	}
}
```

**Step 5: Handle ConfirmBulkRestart in handleConfirmDialogKey**

In `handleConfirmDialogKey`, `case "y", "Y":` switch, add:

```go
		case ConfirmBulkRestart:
			var insts []*session.Instance
			for _, item := range h.flatItems {
				if item.Type == session.ItemTypeSession && item.Session != nil {
					if h.selectedSessionIDs[item.Session.ID] {
						insts = append(insts, item.Session)
					}
				}
			}
			h.confirmDialog.Hide()
			if len(insts) > 0 {
				for _, inst := range insts {
					h.resumingSessions[inst.ID] = time.Now()
				}
				return h, h.bulkRestartSessions(insts)
			}
			return h, nil
```

**Step 6: Handle bulkRestartedMsg in Update**

Near the `sessionRestartedMsg` handler, add:

```go
		case bulkRestartedMsg:
			h.bulkSelectMode = false
			h.selectedSessionIDs = make(map[string]bool)
			if len(msg.errs) > 0 {
				h.setError(fmt.Errorf("restarted %d sessions (%d errors)", len(msg.restartedIDs), len(msg.errs)))
			} else {
				h.setError(fmt.Errorf("restarted %d sessions", len(msg.restartedIDs)))
			}
			return h, nil
```

**Step 7: Run tests**

```
go test ./internal/ui/... -run "TestBulkSelectMode_RKey" -v
```
Expected: PASS

**Step 8: Run full test suite to check for regressions**

```
go test ./internal/ui/... -v 2>&1 | tail -20
```
Expected: all tests pass except the two known pre-existing failures.

**Step 9: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(bulk): bulk restart sessions via R key in visual mode"
```

---

## Task 9: Final smoke test and build verification

**Step 1: Full build**

```
go build ./...
```
Expected: no errors.

**Step 2: Full test suite**

```
go test ./... 2>&1 | grep -E "FAIL|ok|PASS"
```
Expected: all packages pass except the two known `internal/ui` failures.

**Step 3: Manual smoke test (optional)**

```
go run ./cmd/hangar
```
- Press `V` — verify mode banner appears and `□` checkboxes show
- Navigate with `j/k`, press `space` — verify `☑` toggles
- Press `Esc` — verify mode exits, checkboxes disappear
- Press `V`, select sessions, press `d` — verify bulk confirm dialog
- Press `n` to cancel

**Step 4: Final commit if any cleanup needed**

```bash
git add -p
git commit -m "fix(bulk): cleanup after smoke test"
```
