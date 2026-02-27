# Worktree Default + Ghost Item Progress Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make worktree creation the default in the new-session dialog, and show a ghost spinner row in the session list while a worktree is being created in the background.

**Architecture:** Add a `pendingWorktrees []pendingWorktreeItem` slice to `Home`. When async worktree creation starts, append an entry; remove it when `worktreeCreatedForNewSessionMsg` or `worktreeCreatedForForkMsg` arrives. `renderSessionList` in `list_renderer.go` appends ghost rows after the real `flatItems` loop — no cursor interaction because cursor is an index into `flatItems` only.

**Tech Stack:** Go, Bubble Tea (`charmbracelet/bubbletea`), lipgloss, `internal/ui` package.

---

### Task 1: Make worktree the default in NewDialog

**Files:**
- Modify: `internal/ui/newdialog.go:98` (init default)
- Modify: `internal/ui/newdialog.go:122` (ShowInGroup reset)
- Test: `internal/ui/newdialog_test.go`

**Step 1: Write the failing test**

Find the existing `TestNewDialog_*` tests for reference. Add to `internal/ui/newdialog_test.go`:

```go
func TestNewDialog_WorktreeEnabledByDefault(t *testing.T) {
    dlg := NewNewDialog()
    if !dlg.IsWorktreeEnabled() {
        t.Error("expected worktreeEnabled=true by default, got false")
    }
}

func TestNewDialog_ShowInGroup_WorktreeEnabledByDefault(t *testing.T) {
    dlg := NewNewDialog()
    dlg.ShowInGroup("mygroup", "My Group", "/tmp")
    if !dlg.IsWorktreeEnabled() {
        t.Error("expected worktreeEnabled=true after ShowInGroup, got false")
    }
}
```

**Step 2: Run tests to verify they fail**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/feature-worktree-progress-default
go test ./internal/ui/... -run TestNewDialog_WorktreeEnabledByDefault -v
go test ./internal/ui/... -run TestNewDialog_ShowInGroup_WorktreeEnabledByDefault -v
```

Expected: FAIL — `expected worktreeEnabled=true by default, got false`

**Step 3: Change the default**

In `internal/ui/newdialog.go`:

Line 98 — change:
```go
worktreeEnabled: false,
```
to:
```go
worktreeEnabled: true,
```

Line 122 — change:
```go
d.worktreeEnabled = false
```
to:
```go
d.worktreeEnabled = true
```

Also add `d.autoBranchFromName()` call right after line 122 so the branch field
auto-populates when the dialog opens (mirrors what `ToggleWorktree()` does):

```go
d.worktreeEnabled = true
d.autoBranchFromName()
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/ui/... -run TestNewDialog_WorktreeEnabledByDefault -v
go test ./internal/ui/... -run TestNewDialog_ShowInGroup_WorktreeEnabledByDefault -v
```

Expected: PASS

**Step 5: Run full test suite — expect known failures only**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok|---"
```

Pre-existing failures to ignore:
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

No new failures should appear.

**Step 6: Commit**

```bash
git add internal/ui/newdialog.go internal/ui/newdialog_test.go
git commit -m "feat(ui): make worktree creation the default in new-session dialog"
```

---

### Task 2: Add pendingWorktreeItem type and struct field

**Files:**
- Modify: `internal/ui/home.go` (struct definition near line 225–230)

**Step 1: Write a test that verifies the struct field exists**

This is a compilation check — the test in Task 3 will cover behavior. Skip a dedicated test here; proceed to implementation.

**Step 2: Add the type and field**

In `internal/ui/home.go`, find the `// Launching animation state` comment block around line 225–230:

```go
// Launching animation state (for newly created sessions)
launchingSessions  map[string]time.Time // sessionID -> creation time
resumingSessions   map[string]time.Time // sessionID -> resume time (for restart/resume)
forkingSessions    map[string]time.Time // sessionID -> fork start time (fork in progress)
animationFrame     int                  // Current frame for spinner animation
```

Add `pendingWorktrees` to that block:

```go
// Launching animation state (for newly created sessions)
launchingSessions  map[string]time.Time  // sessionID -> creation time
resumingSessions   map[string]time.Time  // sessionID -> resume time (for restart/resume)
forkingSessions    map[string]time.Time  // sessionID -> fork start time (fork in progress)
animationFrame     int                   // Current frame for spinner animation
pendingWorktrees   []pendingWorktreeItem // worktrees being created in the background
```

Add the type definition near the other message types (near line 495–535 where `worktreeCreatedForNewSessionMsg` is defined):

```go
// pendingWorktreeItem tracks a worktree being created in the background.
// It is used to show a ghost spinner row in the session list.
type pendingWorktreeItem struct {
    branchName string
    groupPath  string
    startedAt  time.Time
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/ui/...
```

Expected: no errors.

**Step 4: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): add pendingWorktrees slice and pendingWorktreeItem type to Home"
```

---

### Task 3: Lifecycle — add/remove pending worktrees

**Files:**
- Modify: `internal/ui/home.go` (two call sites + two message handlers)

This task replaces the ephemeral `setError("creating worktree…")` calls with persistent ghost tracking.

**Step 1: New-session creation path (~line 2799–2803)**

Find this block:

```go
// Hide dialog and dispatch async worktree creation so the git fetch
// doesn't block the Bubble Tea event loop (large monorepos can take 30-60s).
h.newDialog.Hide()
h.setError(fmt.Errorf("creating worktree '%s'…", branchName))
```

Replace the `setError` line with:

```go
h.pendingWorktrees = append(h.pendingWorktrees, pendingWorktreeItem{
    branchName: branchName,
    groupPath:  groupPath,
    startedAt:  time.Now(),
})
```

**Step 2: Fork dialog creation path (~line 4348–4352)**

Find this similar block in `handleForkDialogKey` (search for `"creating worktree '%s'\u2026"` in the fork path around line 4351):

```go
h.forkDialog.Hide()
h.setError(fmt.Errorf("creating worktree '%s'\u2026", branchName))
```

Replace the `setError` line with:

```go
h.pendingWorktrees = append(h.pendingWorktrees, pendingWorktreeItem{
    branchName: branchName,
    groupPath:  groupPath,
    startedAt:  time.Now(),
})
```

**Step 3: Remove on worktreeCreatedForNewSessionMsg (~line 2317–2325)**

Find the handler:

```go
case worktreeCreatedForNewSessionMsg:
    if msg.err != nil {
        h.setError(fmt.Errorf("failed to create worktree: %w", msg.err))
        return h, nil
    }
    return h, h.createSessionInGroupWithWorktreeAndOptions(...)
```

Add pending removal before both return paths. Helper inline (no separate method needed for a slice this small):

```go
case worktreeCreatedForNewSessionMsg:
    // Remove the ghost row regardless of success/failure
    h.removePendingWorktree(msg.branchName)
    if msg.err != nil {
        h.setError(fmt.Errorf("failed to create worktree: %w", msg.err))
        return h, nil
    }
    return h, h.createSessionInGroupWithWorktreeAndOptions(...)
```

Note: `worktreeCreatedForNewSessionMsg` needs a `branchName` field — check the struct around line 501:

```go
type worktreeCreatedForNewSessionMsg struct {
    name         string
    command      string
    groupPath    string
    worktreePath string
    repoRoot     string
    branchName   string  // ← this should already exist
    toolOptions  json.RawMessage
    err          error
}
```

If `branchName` is already there (it is, per line 2831), use `msg.branchName`. ✓

**Step 4: Remove on worktreeCreatedForForkMsg (~line 2344–2349)**

Find the handler:

```go
case worktreeCreatedForForkMsg:
    if msg.err != nil {
        h.setError(fmt.Errorf("failed to create worktree: %w", msg.err))
        return h, nil
    }
    return h, h.forkSessionCmdWithOptions(msg.source, msg.title, msg.groupPath, msg.opts)
```

Check what fields `worktreeCreatedForForkMsg` has (around line 534). It needs a `branchName`. Looking at the struct around line 534–540 and the return around line 4382:

```go
type worktreeCreatedForForkMsg struct {
    source      *session.Instance
    title       string
    groupPath   string
    opts        *session.ClaudeOptions
    worktreePath string
    err         error
}
```

The `branchName` is not on this struct. Add it. Find the struct definition and add `branchName string`. Then populate it in the return statement around line 4382:

```go
return worktreeCreatedForForkMsg{
    source:       capturedSource,
    title:        capturedTitle,
    groupPath:    capturedGroupPath,
    opts:         capturedOpts,
    worktreePath: capturedWorktreePath,
    branchName:   capturedBranch,    // add this
}
```

Then update the handler:

```go
case worktreeCreatedForForkMsg:
    h.removePendingWorktree(msg.branchName)
    if msg.err != nil {
        h.setError(fmt.Errorf("failed to create worktree: %w", msg.err))
        return h, nil
    }
    return h, h.forkSessionCmdWithOptions(msg.source, msg.title, msg.groupPath, msg.opts)
```

**Step 5: Add the removePendingWorktree helper**

Add this method to `home.go` near the other animation helpers (`hasActiveAnimation` ~line 1418):

```go
// removePendingWorktree removes the first pending worktree entry with the given branch name.
func (h *Home) removePendingWorktree(branchName string) {
    for i, pw := range h.pendingWorktrees {
        if pw.branchName == branchName {
            h.pendingWorktrees = append(h.pendingWorktrees[:i], h.pendingWorktrees[i+1:]...)
            return
        }
    }
}
```

**Step 6: Verify it compiles**

```bash
go build ./internal/ui/...
```

Expected: no errors.

**Step 7: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): track pending worktrees; remove setError status hack"
```

---

### Task 4: Ghost row rendering in renderSessionList

**Files:**
- Modify: `internal/ui/list_renderer.go:23–80` (renderSessionList)
- Modify: `internal/ui/styles.go` (add ghost row style — if not already suitable)

**Step 1: Check available dim/faint styles in styles.go**

```bash
grep -n "Faint\|DimStyle\|dimStyle\|GhostStyle\|ghostStyle" internal/ui/styles.go | head -20
```

`DimStyle` is likely already defined. We'll use it (or create `stylePendingWorktree` if needed).

**Step 2: Add the ghost row style to styles.go (if DimStyle isn't enough)**

If `DimStyle` exists and is appropriate (faint foreground, no background), use it directly and skip this step.

If a distinct style is needed, add near the other session list styles in `internal/ui/styles.go`:

```go
// stylePendingWorktree is used for ghost rows showing worktree creation in progress
var stylePendingWorktree = lipgloss.NewStyle().
    Foreground(ColorTextDim).
    Faint(true)
```

**Step 3: Write a test for the ghost row**

The rendering functions are on `*Home` and require a full `Home` initialization, making unit tests complex. The existing pattern in `internal/ui/home_test.go` uses `newTestHome()`. Add to `internal/ui/list_renderer_test.go` (create if it doesn't exist):

```go
package ui

import (
    "strings"
    "testing"
    "time"
)

func TestRenderSessionList_ShowsPendingWorktree(t *testing.T) {
    h := newTestHome()
    h.pendingWorktrees = []pendingWorktreeItem{
        {branchName: "feat/my-branch", groupPath: "default", startedAt: time.Now()},
    }
    output := h.renderSessionList(80, 20)
    if !strings.Contains(output, "feat/my-branch") {
        t.Errorf("expected ghost row with branch name in session list, got:\n%s", output)
    }
}
```

**Step 4: Run the test to verify it fails**

```bash
go test ./internal/ui/... -run TestRenderSessionList_ShowsPendingWorktree -v
```

Expected: FAIL — branch name not in output.

**Step 5: Implement ghost row rendering**

In `internal/ui/list_renderer.go`, find `renderSessionList`. After the main `flatItems` loop (just before the final `return b.String()`), add:

```go
// Render ghost rows for pending worktree creations.
// These are intentionally outside flatItems so the cursor never lands on them.
spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
spinner := spinnerFrames[h.animationFrame%len(spinnerFrames)]
for _, pw := range h.pendingWorktrees {
    if visibleCount >= maxVisible {
        break
    }
    ghostRow := DimStyle.Render(fmt.Sprintf("  %s Creating worktree: %s", spinner, pw.branchName))
    b.WriteString(ghostRow)
    b.WriteString("\n")
    visibleCount++
}
```

Note: `DimStyle` is used here — check its import/definition. It should already be imported since it's used elsewhere in the file (see line 61: `DimStyle.Render(...)`).

**Step 6: Run the test to verify it passes**

```bash
go test ./internal/ui/... -run TestRenderSessionList_ShowsPendingWorktree -v
```

Expected: PASS

**Step 7: Run full test suite**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok|---"
```

Expected: only the two pre-existing failures.

**Step 8: Build the binary and smoke test**

```bash
go build ./cmd/hangar/
```

Expected: no errors.

**Step 9: Commit**

```bash
git add internal/ui/list_renderer.go internal/ui/styles.go internal/ui/list_renderer_test.go
git commit -m "feat(ui): render ghost spinner row for pending worktree creation"
```

---

### Task 5: Final validation

**Step 1: Run full test suite with race detector**

```bash
go test -race ./... 2>&1 | grep -E "FAIL|ok|DATA RACE|---"
```

Expected: clean (only the two pre-existing failures).

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit if any cleanup needed, otherwise done**

```bash
git log --oneline -5
```

Should show 4 commits from this feature:
1. `feat(ui): make worktree creation the default in new-session dialog`
2. `feat(ui): add pendingWorktrees slice and pendingWorktreeItem type to Home`
3. `feat(ui): track pending worktrees; remove setError status hack`
4. `feat(ui): render ghost spinner row for pending worktree creation`
