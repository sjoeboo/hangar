# PR Status in Worktree Cleanup Dialog — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show PR number, state, title, and CI check summary inside `WorktreeFinishDialog`; route `d` on worktree sessions to that dialog instead of the simple `ConfirmDialog`.

**Architecture:** Add `prEntry`/`prLoaded` fields + `SetPR()` method to `WorktreeFinishDialog`, render a PR section in `viewOptions()`, wire PR data from `Home`'s cache into the dialog at show-time and on `prFetchedMsg`, and redirect the `d` key for worktree sessions to the same dialog used by `W`.

**Tech Stack:** Go, Bubble Tea (charmbracelet/bubbletea), lipgloss

---

### Task 1: Add PR fields and `SetPR` method to `WorktreeFinishDialog`

**Files:**
- Modify: `internal/ui/worktree_finish_dialog.go:13-34` (struct) and `:42-55` (Show), `:57-62` (Hide)

**Step 1: Write the failing test**

Create `internal/ui/worktree_finish_dialog_test.go`:

```go
package ui

import (
	"strings"
	"testing"
)

func TestWorktreeFinishDialog_SetPR_NoPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(nil, true) // loaded but no PR

	view := d.View()
	if !strings.Contains(view, "No PR") {
		t.Errorf("expected 'No PR' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_Checking(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	// prLoaded=false = still checking
	view := d.View()
	if !strings.Contains(view, "checking") {
		t.Errorf("expected 'checking' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_WithPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(&prCacheEntry{
		Number:       42,
		Title:        "Add user auth",
		State:        "OPEN",
		ChecksPassed: 5,
		ChecksFailed: 1,
		HasChecks:    true,
	}, true)

	view := d.View()
	if !strings.Contains(view, "#42") {
		t.Errorf("expected '#42' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Add user auth") {
		t.Errorf("expected title in view, got:\n%s", view)
	}
	if !strings.Contains(view, "OPEN") {
		t.Errorf("expected 'OPEN' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_CILine(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(&prCacheEntry{
		Number:        7,
		Title:         "Fix bug",
		State:         "OPEN",
		ChecksPassed:  3,
		ChecksFailed:  0,
		ChecksPending: 2,
		HasChecks:     true,
	}, true)

	view := d.View()
	// Should show passed and pending but omit zero failed
	if !strings.Contains(view, "3") {
		t.Errorf("expected passed count in view, got:\n%s", view)
	}
	if !strings.Contains(view, "2") {
		t.Errorf("expected pending count in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_ShowResetsPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetPR(&prCacheEntry{Number: 1, State: "OPEN"}, true)

	// Re-show should reset PR state to "checking"
	d.Show("id2", "other", "feat/bar", "/repo", "/repo/.worktrees/bar")

	view := d.View()
	if !strings.Contains(view, "checking") {
		t.Errorf("expected 'checking' after re-show, got:\n%s", view)
	}
}
```

**Step 2: Run to confirm failure**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_SetPR -v
```
Expected: FAIL — `SetPR` undefined

**Step 3: Add fields and method to `worktree_finish_dialog.go`**

Add two fields to the struct after `errorMsg`:
```go
// PR data (set from Home's cache via SetPR)
prEntry  *prCacheEntry // nil = no PR found
prLoaded bool         // false = still fetching
```

In `Show()`, reset them after existing resets:
```go
d.prEntry = nil
d.prLoaded = false
```

In `Hide()`, reset them:
```go
d.prEntry = nil
d.prLoaded = false
```

Add the new method after `SetExecuting`:
```go
// SetPR updates the PR data shown in the options view.
// Call with loaded=false (default on Show) to show "checking...",
// loaded=true+nil pr to show "No PR found", loaded=true+non-nil for full info.
func (d *WorktreeFinishDialog) SetPR(pr *prCacheEntry, loaded bool) {
	d.prEntry = pr
	d.prLoaded = loaded
}
```

**Step 4: Run tests to verify they pass (except rendering tests)**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_SetPR -v
```
Expected: `TestWorktreeFinishDialog_ShowResetsPR` passes; rendering tests still FAIL (no view changes yet)

**Step 5: Commit**

```bash
git add internal/ui/worktree_finish_dialog.go internal/ui/worktree_finish_dialog_test.go
git commit -m "feat: add SetPR method and PR fields to WorktreeFinishDialog"
```

---

### Task 2: Render the PR section in `viewOptions()`

**Files:**
- Modify: `internal/ui/worktree_finish_dialog.go:185-234` (`viewOptions`)

**Step 1: Write the rendering test** (already written in Task 1 test file — `TestWorktreeFinishDialog_SetPR_NoPR`, `TestWorktreeFinishDialog_SetPR_WithPR`, `TestWorktreeFinishDialog_SetPR_CILine`)

**Step 2: Run to confirm they still fail**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_SetPR_NoPR -v
```
Expected: FAIL — "No PR" not in view

**Step 3: Add PR section to `viewOptions()`**

In `viewOptions()`, after the dirty-status block (after the `b.WriteString("\n\n")` following the status section), add before the keep-branch checkbox:

```go
// PR status
b.WriteString(labelStyle.Render("  PR:       "))
if !d.prLoaded {
	b.WriteString(labelStyle.Render("checking..."))
} else if d.prEntry == nil {
	b.WriteString(labelStyle.Render("No PR found"))
} else {
	pr := d.prEntry
	var stateStyle lipgloss.Style
	switch pr.State {
	case "OPEN":
		stateStyle = PRBadgeOpen
	case "MERGED":
		stateStyle = PRBadgeMerged
	case "CLOSED":
		stateStyle = PRBadgeClosed
	default: // DRAFT, unknown
		stateStyle = labelStyle
	}
	b.WriteString(valueStyle.Render(fmt.Sprintf("#%d", pr.Number)))
	b.WriteString(labelStyle.Render(" · "))
	b.WriteString(stateStyle.Render(pr.State))
	b.WriteString(labelStyle.Render(" · "))
	// Truncate title to fit dialog width
	title := pr.Title
	maxTitle := dialogWidth - 20
	if maxTitle > 0 && len(title) > maxTitle {
		title = title[:maxTitle] + "…"
	}
	b.WriteString(valueStyle.Render(fmt.Sprintf("%q", title)))
	b.WriteString("\n")

	// CI check line (only if checks exist; omit zero counts)
	if pr.HasChecks {
		b.WriteString(labelStyle.Render("             "))
		ciParts := []string{}
		if pr.ChecksPassed > 0 {
			ciParts = append(ciParts, lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("✓ %d passed", pr.ChecksPassed)))
		}
		if pr.ChecksFailed > 0 {
			ciParts = append(ciParts, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✗ %d failed", pr.ChecksFailed)))
		}
		if pr.ChecksPending > 0 {
			ciParts = append(ciParts, lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("⟳ %d pending", pr.ChecksPending)))
		}
		b.WriteString(strings.Join(ciParts, "  "))
	}
}
b.WriteString("\n\n")
```

Note: remove the existing `b.WriteString("\n\n")` that currently separates the status block from the checkbox, since the PR section now ends with `\n\n`.

**Step 4: Run all rendering tests**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_SetPR -v
```
Expected: all PASS

**Step 5: Run full UI test suite to catch regressions**

```bash
go test ./internal/ui/... -v 2>&1 | tail -30
```
Expected: only the two pre-existing failing tests (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`) fail

**Step 6: Commit**

```bash
git add internal/ui/worktree_finish_dialog.go
git commit -m "feat: render PR status section in WorktreeFinishDialog options view"
```

---

### Task 3: Wire PR data into dialog on show (W key handler)

**Files:**
- Modify: `internal/ui/home.go:3893-3914` (`"W", "shift+w"` case)

**Step 1: Write a test**

In `internal/ui/home_test.go`, add:

```go
func TestWorktreeFinishDialog_ShowWithCachedPR(t *testing.T) {
	home := newTestHome(t)
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Pre-populate PR cache
	home.prCacheMu.Lock()
	home.prCache[inst.ID] = &prCacheEntry{Number: 55, State: "OPEN", Title: "My PR"}
	home.prCacheTs[inst.ID] = time.Now()
	home.prCacheMu.Unlock()

	// Trigger W key
	home.cursor = 0
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})

	// Dialog should have PR data
	if !home.worktreeFinishDialog.IsVisible() {
		t.Fatal("expected dialog visible")
	}
	if home.worktreeFinishDialog.prEntry == nil {
		t.Error("expected prEntry to be set from cache")
	}
	if home.worktreeFinishDialog.prEntry.Number != 55 {
		t.Errorf("expected PR #55, got %d", home.worktreeFinishDialog.prEntry.Number)
	}
	if !home.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded=true when cache entry exists")
	}
}

func TestWorktreeFinishDialog_ShowWithNoCachedPR(t *testing.T) {
	home := newTestHome(t)
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// No PR in cache

	home.cursor = 0
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})

	if !home.worktreeFinishDialog.IsVisible() {
		t.Fatal("expected dialog visible")
	}
	// prLoaded=false means still checking
	if home.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded=false when no cache entry")
	}
}
```

**Step 2: Find or create `makeWorktreeInstance` helper**

Check `home_test.go` for existing helpers. If there's a `newTestSession` or similar, use that. Otherwise add to test file:

```go
func makeWorktreeInstance(id, branch string) *session.Instance {
	return &session.Instance{
		ID:              id,
		Title:           "test-" + id,
		WorktreePath:    "/tmp/worktrees/" + id,
		WorktreeRepoRoot: "/tmp/repo",
		WorktreeBranch:  branch,
	}
}
```

**Step 3: Run to confirm failure**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_ShowWithCachedPR -v
```
Expected: FAIL — `prEntry` still nil after show

**Step 4: Modify the `"W", "shift+w"` handler in `home.go`**

After the `h.worktreeFinishDialog.Show(...)` call (line ~3904), add PR wiring before the dirty-check return:

```go
// Wire in current PR cache entry (if any)
h.prCacheMu.Lock()
cachedPR, hasCachedPR := h.prCache[inst.ID]
_, hasCachedTS := h.prCacheTs[inst.ID]
h.prCacheMu.Unlock()
if hasCachedTS {
    h.worktreeFinishDialog.SetPR(cachedPR, true)
} // else: prLoaded stays false ("checking...")
```

**Step 5: Run tests**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_ShowWith -v
```
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat: wire PR cache into WorktreeFinishDialog on show"
```

---

### Task 4: Push PR updates to dialog from `prFetchedMsg`

**Files:**
- Modify: `internal/ui/home.go:2826-2855` (`prFetchedMsg` case)

**Step 1: Write a test**

In `internal/ui/home_test.go`, add:

```go
func TestWorktreeFinishDialog_PRFetchedUpdatesDialog(t *testing.T) {
	home := newTestHome(t)
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	// Open dialog (no PR cached yet)
	home.cursor = 0
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
	if home.worktreeFinishDialog.prEntry == nil || home.worktreeFinishDialog.prEntry.Number != 99 {
		t.Error("expected prEntry updated to PR #99")
	}
}

func TestWorktreeFinishDialog_PRFetchedIgnoresDifferentSession(t *testing.T) {
	home := newTestHome(t)
	inst := makeWorktreeInstance("sess1", "feat/test")
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 0
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("W")})

	// Fetch for a different session
	home.Update(prFetchedMsg{
		sessionID: "other-session",
		pr:        &prCacheEntry{Number: 77, State: "OPEN"},
	})

	// Dialog should remain unloaded
	if home.worktreeFinishDialog.prLoaded {
		t.Error("expected prLoaded unchanged for different session")
	}
}
```

**Step 2: Run to confirm failure**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_PRFetched -v
```
Expected: FAIL

**Step 3: Modify `prFetchedMsg` handler in `home.go`**

After the existing cache update block (after `h.prCacheMu.Unlock()` on line ~2831) and before the todo status block, add:

```go
// If WorktreeFinishDialog is open for this session, push updated PR data
if h.worktreeFinishDialog.IsVisible() && h.worktreeFinishDialog.GetSessionID() == msg.sessionID {
    h.worktreeFinishDialog.SetPR(msg.pr, true)
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestWorktreeFinishDialog_PRFetched -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat: push PR updates to WorktreeFinishDialog when open"
```

---

### Task 5: Route `d` key on worktree sessions to `WorktreeFinishDialog`

**Files:**
- Modify: `internal/ui/home.go:4049-4059` (`case "d":`)

**Step 1: Write a test**

In `internal/ui/home_test.go`, add:

```go
func TestDeleteKey_WorktreeSessionOpensFinishDialog(t *testing.T) {
	home := newTestHome(t)
	inst := makeWorktreeInstance("sess1", "feat/delete-me")
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 0
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if !home.worktreeFinishDialog.IsVisible() {
		t.Error("expected WorktreeFinishDialog to be visible when d pressed on worktree session")
	}
	if home.confirmDialog.IsVisible() {
		t.Error("expected ConfirmDialog NOT visible for worktree session")
	}
}

func TestDeleteKey_NonWorktreeSessionOpensConfirmDialog(t *testing.T) {
	home := newTestHome(t)
	inst := &session.Instance{ID: "sess2", Title: "plain-session"}
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.rebuildFlatItems()

	home.cursor = 0
	home.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if home.worktreeFinishDialog.IsVisible() {
		t.Error("expected WorktreeFinishDialog NOT visible for non-worktree session")
	}
	if !home.confirmDialog.IsVisible() {
		t.Error("expected ConfirmDialog to be visible for non-worktree session")
	}
}
```

**Step 2: Run to confirm failure**

```bash
go test ./internal/ui/... -run TestDeleteKey_ -v
```
Expected: `TestDeleteKey_WorktreeSessionOpensFinishDialog` FAIL (opens ConfirmDialog instead)

**Step 3: Modify `case "d":` in `home.go`**

Replace:
```go
case "d":
    // Show confirmation dialog before deletion (prevents accidental deletion)
    if h.cursor < len(h.flatItems) {
        item := h.flatItems[h.cursor]
        if item.Type == session.ItemTypeSession && item.Session != nil {
            h.confirmDialog.ShowDeleteSession(item.Session.ID, item.Session.Title)
        } else if item.Type == session.ItemTypeGroup && item.Path != session.DefaultGroupPath {
            h.confirmDialog.ShowDeleteGroup(item.Path, item.Group.Name)
        }
    }
    return h, nil
```

With:
```go
case "d":
    // Show confirmation dialog before deletion (prevents accidental deletion)
    if h.cursor < len(h.flatItems) {
        item := h.flatItems[h.cursor]
        if item.Type == session.ItemTypeSession && item.Session != nil {
            inst := item.Session
            if inst.IsWorktree() {
                // Worktree sessions get the full finish dialog (same as W)
                h.worktreeFinishDialog.SetSize(h.width, h.height)
                h.worktreeFinishDialog.Show(inst.ID, inst.Title, inst.WorktreeBranch, inst.WorktreeRepoRoot, inst.WorktreePath)
                // Wire in current PR cache
                h.prCacheMu.Lock()
                cachedPR, hasCachedPR := h.prCache[inst.ID]
                _, hasCachedTS := h.prCacheTs[inst.ID]
                h.prCacheMu.Unlock()
                if hasCachedTS {
                    h.worktreeFinishDialog.SetPR(cachedPR, true)
                }
                // Trigger async dirty check
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

Note: There's a declared-but-not-used `hasCachedPR` — use `_ = hasCachedPR` or restructure. Use this instead:
```go
h.prCacheMu.Lock()
cachedPR, hasCachedTS := h.prCache[inst.ID]
_, hasTS := h.prCacheTs[inst.ID]
h.prCacheMu.Unlock()
if hasTS {
    h.worktreeFinishDialog.SetPR(cachedPR, true)
}
```
Wait — `h.prCache[inst.ID]` returns `*prCacheEntry, bool` via map lookup. The bool tells us if the key exists. Since we set `prCacheTs` at the same time as `prCache`, checking either is equivalent. Use the cache map itself:

```go
h.prCacheMu.Lock()
cachedPR, hasPRCached := h.prCache[inst.ID]
h.prCacheMu.Unlock()
if hasPRCached {
    h.worktreeFinishDialog.SetPR(cachedPR, true)
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestDeleteKey_ -v
```
Expected: PASS

**Step 5: Run full suite**

```bash
go test ./internal/ui/... 2>&1 | tail -20
```
Expected: only the two pre-existing failures

**Step 6: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat: route delete key on worktree sessions to WorktreeFinishDialog"
```

---

### Task 6: Final build verification

**Step 1: Build**

```bash
go build ./...
```
Expected: no errors

**Step 2: Run all tests**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok|---"
```
Expected: all packages `ok` (except the two known pre-existing UI failures)

**Step 3: Verify the two known failures are unchanged**

```bash
go test ./internal/ui/... -run "TestNewDialog_WorktreeToggle_ViaKeyPress|TestNewDialog_TypingResetsSuggestionNavigation" -v
```
Expected: same failures as before this work (not new regressions)
