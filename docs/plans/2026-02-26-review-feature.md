# Review Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `v` key in the Hangar sidebar that opens a "Review PR" dialog — the user enters a PR # or branch name, Hangar fetches the branch, creates a worktree, launches Claude, and delivers a `/pr-review` prompt automatically.

**Architecture:** A new `ReviewDialog` struct (same two-step pattern as `WorktreeFinishDialog`) collects a PR # or branch name; home.go handles async resolution via `gh pr view`, then reuses existing `git.FetchBranch` + `git.CreateWorktree` + `createSessionInGroupWithWorktreeAndOptions` machinery; after session start a delayed `tea.Cmd` delivers the `/pr-review` prompt as tmux keystrokes.

**Tech Stack:** Go, Bubble Tea (bubbletea), lipgloss, textinput (charmbracelet), tmux, `gh` CLI for PR resolution.

---

## Overview of files touched

| File | Change |
|------|--------|
| `internal/git/git.go` | Add `FetchBranch()` helper |
| `internal/git/git_test.go` | Add tests for `FetchBranch` |
| `internal/session/instance.go` | Add `SendText()` method |
| `internal/ui/review_dialog.go` | New — `ReviewDialog` struct |
| `internal/ui/review_dialog_test.go` | New — dialog unit tests |
| `internal/ui/home.go` | 7-point wiring + `v` key + message handlers |

---

## Task 1: Add `FetchBranch` to `internal/git/git.go`

Fetches a remote branch as a local tracking branch so `CreateWorktree` can check it out.

**Files:**
- Modify: `internal/git/git.go`
- Modify: `internal/git/git_test.go`

**Step 1: Write the failing test**

Open `internal/git/git_test.go`. Add at the end:

```go
func TestFetchBranch_InvalidBranch(t *testing.T) {
    dir := t.TempDir()
    // Bare git repo with no remote — fetch must fail gracefully
    if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
        t.Fatalf("git init: %s %v", out, err)
    }
    err := FetchBranch(dir, "nonexistent")
    if err == nil {
        t.Fatal("expected error fetching from repo with no remote, got nil")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/git/... -run TestFetchBranch -v
```

Expected: FAIL — `FetchBranch` undefined.

**Step 3: Implement `FetchBranch`**

Open `internal/git/git.go`. Add after `UpdateBaseBranch` (around line 466):

```go
// FetchBranch fetches a single remote branch as a local tracking branch.
// This is used before creating a review worktree so the branch exists locally.
// Runs: git fetch origin <branch>:<branch>
func FetchBranch(repoDir, branch string) error {
    const timeout = 60 * time.Second
    const waitDelay = 5 * time.Second

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "fetch", "origin",
        branch+":"+branch)
    cmd.WaitDelay = waitDelay
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("fetch branch %q failed: %s: %w",
            branch, strings.TrimSpace(string(output)), err)
    }
    return nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/git/... -run TestFetchBranch -v
```

Expected: PASS

**Step 5: Build check**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat(git): add FetchBranch helper for review worktrees"
```

---

## Task 2: Add `SendText` to `internal/session/instance.go`

Needed to deliver the `/pr-review` prompt to a running Claude session.

**Files:**
- Modify: `internal/session/instance.go`

**Step 1: Check if a send method already exists**

```bash
grep -n "SendText\|SendKeys\|sendText\|sendKeys" internal/session/instance.go
```

If a method already exists that sends keystrokes to the tmux pane, skip to Step 5 and note the real method name for later tasks. Otherwise continue.

**Step 2: Find where `tmuxSession` is used in instance.go**

```bash
grep -n "tmuxSession\." internal/session/instance.go | head -20
```

Note the method calls available on `*tmux.Session` (e.g. `SendKeysAndEnter`).

**Step 3: Find `SendKeysAndEnter` in tmux.go**

```bash
grep -n "func.*SendKeysAndEnter\|func.*SendKeys" internal/tmux/tmux.go
```

Note the exact signature.

**Step 4: Add `SendText` to instance.go**

Find the `IsWorktree()` method (around line 209). Add after it:

```go
// SendText sends text to the session's tmux pane as if the user typed it.
// Used to deliver initial prompts (e.g. /pr-review) after session start.
func (i *Instance) SendText(text string) error {
    if i.tmuxSession == nil {
        return fmt.Errorf("no tmux session")
    }
    return i.tmuxSession.SendKeysAndEnter(text)
}
```

> If Step 3 revealed a different method name (not `SendKeysAndEnter`), use the correct one here.

**Step 5: Build check**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add internal/session/instance.go
git commit -m "feat(session): add SendText method for post-start prompt delivery"
```

---

## Task 3: Create `ReviewDialog` struct

New file. Two-step dialog: input → confirm (with optional async PR resolution).

**Files:**
- Create: `internal/ui/review_dialog.go`
- Create: `internal/ui/review_dialog_test.go`

**Step 1: Write the failing tests first**

Create `internal/ui/review_dialog_test.go`:

```go
package ui

import (
    "testing"
)

func TestReviewDialog_InitiallyHidden(t *testing.T) {
    d := NewReviewDialog()
    if d.IsVisible() {
        t.Fatal("expected dialog to be hidden initially")
    }
}

func TestReviewDialog_ShowMakesVisible(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    if !d.IsVisible() {
        t.Fatal("expected dialog to be visible after Show")
    }
}

func TestReviewDialog_EscHidesDialog(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    action := d.HandleKey("esc")
    if action != "cancel" {
        t.Fatalf("expected action 'cancel', got %q", action)
    }
    if d.IsVisible() {
        t.Fatal("expected dialog to be hidden after esc")
    }
}

func TestReviewDialog_EnterWithEmptyInputDoesNothing(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    action := d.HandleKey("enter")
    if action != "" {
        t.Fatalf("expected empty action for enter with empty input, got %q", action)
    }
    if !d.IsVisible() {
        t.Fatal("expected dialog to remain visible")
    }
}

func TestReviewDialog_InputDetectsPRNumber(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    // Simulate typing "42"
    for _, r := range "42" {
        d.HandleKey(string(r))
    }
    if !d.IsPRInput() {
        t.Fatal("expected '42' to be detected as a PR number")
    }
}

func TestReviewDialog_InputDetectsBranchName(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    for _, r := range "feature/my-branch" {
        d.HandleKey(string(r))
    }
    if d.IsPRInput() {
        t.Fatal("expected 'feature/my-branch' to be detected as a branch, not a PR number")
    }
}

func TestReviewDialog_SetResolvedMovesToConfirmStep(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
    if d.step != 1 {
        t.Fatalf("expected step 1 after SetResolved, got %d", d.step)
    }
}

func TestReviewDialog_ConfirmStepEnterReturnsConfirm(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
    action := d.HandleKey("enter")
    if action != "confirm" {
        t.Fatalf("expected action 'confirm', got %q", action)
    }
}

func TestReviewDialog_ConfirmStepEscGoesBack(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
    action := d.HandleKey("esc")
    if action != "" {
        t.Fatalf("expected empty action for esc in confirm step, got %q", action)
    }
    if d.step != 0 {
        t.Fatalf("expected step 0 after esc in confirm step, got %d", d.step)
    }
}

func TestReviewDialog_GetReviewValues(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")

    branch, prNum, sessionName, initialPrompt := d.GetReviewValues()
    if branch != "feature/auth-fix" {
        t.Errorf("branch: want %q got %q", "feature/auth-fix", branch)
    }
    if prNum != "42" {
        t.Errorf("prNum: want %q got %q", "42", prNum)
    }
    if sessionName != "review/pr-42" {
        t.Errorf("sessionName: want %q got %q", "review/pr-42", sessionName)
    }
    if initialPrompt != "/pr-review 42" {
        t.Errorf("initialPrompt: want %q got %q", "/pr-review 42", initialPrompt)
    }
}

func TestReviewDialog_GetReviewValuesForBranch(t *testing.T) {
    d := NewReviewDialog()
    d.Show("hangar", "/home/user/code/hangar")
    d.SetSize(120, 40)
    d.SetResolved("feature/my-branch", "", false, "")

    branch, prNum, sessionName, initialPrompt := d.GetReviewValues()
    if branch != "feature/my-branch" {
        t.Errorf("branch: want %q got %q", "feature/my-branch", branch)
    }
    if prNum != "" {
        t.Errorf("prNum: want empty got %q", prNum)
    }
    if sessionName != "review/feature/my-branch" {
        t.Errorf("sessionName: want %q got %q", "review/feature/my-branch", sessionName)
    }
    if initialPrompt != "/pr-review feature/my-branch" {
        t.Errorf("initialPrompt: want %q got %q", "/pr-review feature/my-branch", initialPrompt)
    }
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/... -run TestReviewDialog -v
```

Expected: FAIL — `NewReviewDialog` undefined.

**Step 3: Create `internal/ui/review_dialog.go`**

```go
package ui

import (
    "fmt"
    "strings"
    "unicode"

    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// ReviewDialog is a two-step dialog for starting a Claude-assisted PR review session.
//
// Step 0: Input — user types a PR # (digits) or branch name.
// Step 1: Confirm — shows resolved branch, session name, and /pr-review prompt before creating.
type ReviewDialog struct {
    visible bool
    width   int
    height  int

    // Project context (set on Show)
    projectName string
    repoDir     string

    // Input (step 0)
    input     textinput.Model
    resolving bool // true while async gh lookup is in flight
    errorMsg  string

    // Resolved values (set by SetResolved, moves dialog to step 1)
    step           int
    resolvedBranch string
    prTitle        string
    isPR           bool
    prNumber       string
}

func NewReviewDialog() *ReviewDialog {
    ti := textinput.New()
    ti.Placeholder = "PR # or branch name"
    ti.CharLimit = 200
    return &ReviewDialog{input: ti}
}

func (d *ReviewDialog) Show(projectName, repoDir string) {
    d.visible = true
    d.projectName = projectName
    d.repoDir = repoDir
    d.step = 0
    d.resolving = false
    d.errorMsg = ""
    d.resolvedBranch = ""
    d.prTitle = ""
    d.isPR = false
    d.prNumber = ""
    d.input.Reset()
    d.input.Focus()
}

func (d *ReviewDialog) Hide() {
    d.visible = false
}

func (d *ReviewDialog) IsVisible() bool {
    return d.visible
}

func (d *ReviewDialog) SetSize(width, height int) {
    d.width = width
    d.height = height
    d.input.Width = min(60, width-20)
}

// SetResolving is called when async PR resolution starts.
func (d *ReviewDialog) SetResolving(resolving bool) {
    d.resolving = resolving
    d.errorMsg = ""
}

// SetResolved is called when async PR resolution completes successfully.
// It advances the dialog to the confirm step.
func (d *ReviewDialog) SetResolved(branch, title string, isPR bool, prNum string) {
    d.resolving = false
    d.resolvedBranch = branch
    d.prTitle = title
    d.isPR = isPR
    d.prNumber = prNum
    d.errorMsg = ""
    d.step = 1
}

// SetError displays an inline error (e.g. gh lookup failed).
func (d *ReviewDialog) SetError(msg string) {
    d.resolving = false
    d.errorMsg = msg
}

// IsPRInput returns true if the current input text looks like a PR number (all digits).
func (d *ReviewDialog) IsPRInput() bool {
    v := strings.TrimSpace(d.input.Value())
    if v == "" {
        return false
    }
    for _, r := range v {
        if !unicode.IsDigit(r) {
            return false
        }
    }
    return true
}

// GetRawInput returns the trimmed input value.
func (d *ReviewDialog) GetRawInput() string {
    return strings.TrimSpace(d.input.Value())
}

// GetRepoDir returns the project repository root.
func (d *ReviewDialog) GetRepoDir() string {
    return d.repoDir
}

// GetReviewValues returns the confirmed review parameters after SetResolved has been called.
// Returns (branch, prNumber, sessionName, initialPrompt).
func (d *ReviewDialog) GetReviewValues() (branch, prNum, sessionName, initialPrompt string) {
    branch = d.resolvedBranch
    prNum = d.prNumber
    if d.isPR {
        sessionName = "review/pr-" + d.prNumber
        initialPrompt = "/pr-review " + d.prNumber
    } else {
        sessionName = "review/" + d.resolvedBranch
        initialPrompt = "/pr-review " + d.resolvedBranch
    }
    return
}

// HandleKey processes a key event. Returns an action string:
//   - "resolve" — step 0 enter with non-empty input (home.go should kick off resolution)
//   - "confirm" — step 1 enter (home.go should create the session)
//   - "cancel"  — esc at step 0 (dialog hides itself)
//   - ""        — no action needed
func (d *ReviewDialog) HandleKey(key string) string {
    if d.resolving {
        return "" // block input while resolving
    }

    switch key {
    case "esc":
        if d.step == 1 {
            // Go back to input step
            d.step = 0
            d.errorMsg = ""
            d.input.Focus()
            return ""
        }
        d.Hide()
        return "cancel"

    case "enter":
        if d.step == 0 {
            v := strings.TrimSpace(d.input.Value())
            if v == "" {
                return ""
            }
            d.errorMsg = ""
            return "resolve"
        }
        if d.step == 1 {
            return "confirm"
        }

    default:
        if d.step == 0 {
            var cmd tea.Cmd
            d.input, cmd = d.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
            _ = cmd
            // Clear error on typing
            d.errorMsg = ""
        }
    }
    return ""
}

// Update handles bubbletea messages (needed for textinput cursor blink etc.)
func (d *ReviewDialog) Update(msg tea.Msg) tea.Cmd {
    if !d.visible || d.step != 0 {
        return nil
    }
    var cmd tea.Cmd
    d.input, cmd = d.input.Update(msg)
    return cmd
}

func (d *ReviewDialog) View() string {
    if !d.visible {
        return ""
    }

    dialogWidth := min(60, d.width-4)
    if dialogWidth < 30 {
        dialogWidth = 30
    }

    var content string
    switch d.step {
    case 0:
        content = d.viewInputStep(dialogWidth)
    case 1:
        content = d.viewConfirmStep(dialogWidth)
    }

    box := dialogBorderStyle.
        Width(dialogWidth).
        Render(content)

    return lipgloss.Place(d.width, d.height,
        lipgloss.Center, lipgloss.Center,
        box)
}

func (d *ReviewDialog) viewInputStep(width int) string {
    title := dialogTitleStyle.Render("Review PR")

    projectLine := dialogLabelStyle.Render("Project: ") +
        dialogValueStyle.Render(d.projectName)

    inputLine := d.input.View()

    var statusLine string
    switch {
    case d.resolving:
        statusLine = dialogSubtleStyle.Render("Resolving PR…")
    case d.errorMsg != "":
        statusLine = dialogErrorStyle.Render(d.errorMsg)
    default:
        statusLine = dialogSubtleStyle.Render("enter to continue · esc to cancel")
    }

    return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
        title, projectLine, inputLine, statusLine)
}

func (d *ReviewDialog) viewConfirmStep(width int) string {
    title := dialogTitleStyle.Render("Review PR")

    var lines []string
    lines = append(lines, dialogLabelStyle.Render("Project: ")+
        dialogValueStyle.Render(d.projectName))

    if d.isPR && d.prTitle != "" {
        lines = append(lines, dialogLabelStyle.Render(fmt.Sprintf("PR #%s: ", d.prNumber))+
            dialogValueStyle.Render(d.prTitle))
    }

    lines = append(lines, dialogLabelStyle.Render("Branch:  ")+
        dialogValueStyle.Render(d.resolvedBranch))

    _, _, sessionName, _ := d.GetReviewValues()
    lines = append(lines, dialogLabelStyle.Render("Session: ")+
        dialogValueStyle.Render(sessionName))

    var statusLine string
    if d.errorMsg != "" {
        statusLine = dialogErrorStyle.Render(d.errorMsg)
    } else {
        statusLine = dialogSubtleStyle.Render("enter to create · esc to go back")
    }

    return fmt.Sprintf("%s\n\n%s\n\n%s",
        title,
        strings.Join(lines, "\n"),
        statusLine)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

> **Note on styles**: `dialogBorderStyle`, `dialogTitleStyle`, `dialogLabelStyle`, `dialogValueStyle`, `dialogSubtleStyle`, `dialogErrorStyle` — search `internal/ui/styles.go` for the existing style variables used in other dialogs (e.g. `WorktreeFinishDialog`). Use those same variables rather than defining new ones. If a style doesn't exist, define it in `styles.go`. Also check if `min()` is already defined in the package; if so remove the duplicate.

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestReviewDialog -v
```

Expected: PASS for all `TestReviewDialog_*` tests. The two known-failing tests (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`) are pre-existing failures — ignore them.

**Step 5: Build check**

```bash
go build ./...
```

Fix any compilation errors (most likely style variable names or the `min` duplicate).

**Step 6: Commit**

```bash
git add internal/ui/review_dialog.go internal/ui/review_dialog_test.go
git commit -m "feat(ui): add ReviewDialog for PR review session creation"
```

---

## Task 4: Wire `ReviewDialog` into `home.go` — struct, init, resize, view, message types

**Files:**
- Modify: `internal/ui/home.go`

### Step 1: Add struct field

Find the `Home` struct definition (around line 114). Find where `worktreeFinishDialog *WorktreeFinishDialog` is declared (around line 144). Add immediately after it:

```go
reviewDialog *ReviewDialog
```

### Step 2: Initialize in constructor

Find `NewHomeWithProfileAndMode` (around line 546). Find where `worktreeFinishDialog: NewWorktreeFinishDialog(),` is initialized. Add immediately after:

```go
reviewDialog: NewReviewDialog(),
```

### Step 3: Add to SetSize / windowResized

Find the bulk `SetSize` calls (around line 5883). Find `h.worktreeFinishDialog.SetSize(h.width, h.height)`. Add immediately after:

```go
h.reviewDialog.SetSize(h.width, h.height)
```

### Step 4: Add to View()

Find the dialog visibility checks in `View()` (around line 5974). Find:

```go
if h.worktreeFinishDialog.IsVisible() {
    return h.worktreeFinishDialog.View()
}
```

Add immediately after:

```go
if h.reviewDialog.IsVisible() {
    return h.reviewDialog.View()
}
```

### Step 5: Add new message types

Find the block of message type definitions (around lines 369–531). Add after the `worktreeCreatedForNewSessionMsg` block:

```go
// reviewPRResolvedMsg is returned when the async gh pr view lookup completes.
type reviewPRResolvedMsg struct {
    branch string
    title  string
    isPR   bool
    prNum  string
    err    error
}

// reviewSessionCreatedMsg is returned when the review worktree and session are ready.
type reviewSessionCreatedMsg struct {
    instance      *session.Instance
    initialPrompt string
    err           error
}

// reviewPromptSentMsg is returned after the /pr-review prompt is delivered.
type reviewPromptSentMsg struct{}
```

### Step 6: Build check

```bash
go build ./...
```

### Step 7: Commit

```bash
git add internal/ui/home.go
git commit -m "feat(ui): wire ReviewDialog into Home struct (7-point wiring)"
```

---

## Task 5: Add `v` key handler in `home.go`

**Files:**
- Modify: `internal/ui/home.go`

### Step 1: Find where to add the key handler

Find the `W` / `shift+w` key handler (around line 4190). It's inside the main key switch in `Update()`. Add the `v` handler in the same switch block, near the other single-letter session keys:

```go
case "v":
    // Open review dialog for the current project
    repoDir, projectName, groupPath := h.getReviewContext()
    if repoDir == "" {
        h.setError(fmt.Errorf("no project found — select a session or project group first"))
        return h, nil
    }
    h.reviewDialog.SetSize(h.width, h.height)
    h.reviewDialog.Show(projectName, repoDir)
    h.reviewDialog.groupPath = groupPath
    return h, nil
```

### Step 2: Add `getReviewContext` helper method

Find `func (h *Home) finishWorktree(` (around line 9422). Add a new helper method before it:

```go
// getReviewContext derives the project repo directory, display name, and group path
// from the currently selected sidebar item. Returns empty strings if no project context
// can be determined.
func (h *Home) getReviewContext() (repoDir, projectName, groupPath string) {
    if h.cursor >= len(h.flatItems) {
        return "", "", ""
    }
    item := h.flatItems[h.cursor]

    switch item.Type {
    case session.ItemTypeSession:
        inst := item.Session
        if inst == nil || inst.ProjectPath == "" {
            return "", "", ""
        }
        // Use worktree repo root if this is already a worktree session
        dir := inst.ProjectPath
        if inst.WorktreeRepoRoot != "" {
            dir = inst.WorktreeRepoRoot
        }
        name := inst.GroupPath
        if name == "" {
            name = inst.Title
        }
        return dir, name, inst.GroupPath

    case session.ItemTypeGroup:
        groupName := item.Group
        if groupName == "" {
            return "", "", ""
        }
        // Look up project by name in projects.toml
        proj, err := session.GetProject(groupName)
        if err != nil || proj == nil {
            return "", "", ""
        }
        return proj.BaseDir, proj.Name, groupName
    }

    return "", "", ""
}
```

> **Note on `item.Group`**: Check the actual field name for the group name on `session.Item` by running `grep -n "ItemTypeGroup\|\.Group\b\|GroupName\|GroupPath" internal/session/` — use whatever the real field is named.

### Step 3: Add `groupPath` field to ReviewDialog

Open `internal/ui/review_dialog.go`. Add `groupPath string` to the `ReviewDialog` struct:

```go
type ReviewDialog struct {
    // ... existing fields ...
    groupPath string // set by home.go before Show
}
```

### Step 4: Build check

```bash
go build ./...
```

Fix any field name issues found in Step 2's note.

### Step 5: Run existing tests

```bash
go test ./internal/ui/... -run TestReviewDialog -v
```

Expected: all `TestReviewDialog_*` PASS.

### Step 6: Commit

```bash
git add internal/ui/home.go internal/ui/review_dialog.go
git commit -m "feat(ui): add 'v' key to open ReviewDialog from project/session context"
```

---

## Task 6: Handle `ReviewDialog` key events and async PR resolution

This wires up the dialog's `HandleKey` return values in `home.go`'s `Update()`.

**Files:**
- Modify: `internal/ui/home.go`

### Step 1: Find where other dialog key events are handled in Update()

Find the block that handles `WorktreeFinishDialog` key events. It will look like:

```go
if h.worktreeFinishDialog.IsVisible() {
    action := h.worktreeFinishDialog.HandleKey(msg.String())
    // ...
    return h, nil
}
```

### Step 2: Add ReviewDialog key handling before it

```go
if h.reviewDialog.IsVisible() {
    action := h.reviewDialog.HandleKey(msg.String())
    switch action {
    case "cancel":
        return h, nil

    case "resolve":
        input := h.reviewDialog.GetRawInput()
        repoDir := h.reviewDialog.GetRepoDir()
        h.reviewDialog.SetResolving(true)
        isPR := h.reviewDialog.IsPRInput()
        return h, func() tea.Msg {
            if isPR {
                // Use gh to resolve PR # to branch name
                branch, title, err := resolvePRBranch(repoDir, input)
                if err != nil {
                    return reviewPRResolvedMsg{err: err}
                }
                return reviewPRResolvedMsg{
                    branch: branch,
                    title:  title,
                    isPR:   true,
                    prNum:  input,
                }
            }
            // Branch name input — no resolution needed
            return reviewPRResolvedMsg{
                branch: input,
                isPR:   false,
            }
        }

    case "confirm":
        branch, prNum, sessionName, initialPrompt := h.reviewDialog.GetReviewValues()
        repoDir := h.reviewDialog.GetRepoDir()
        groupPath := h.reviewDialog.groupPath
        h.reviewDialog.Hide()
        return h, h.createReviewSession(repoDir, branch, prNum, sessionName, groupPath, initialPrompt)
    }
    return h, nil
}
```

### Step 3: Handle `reviewPRResolvedMsg` in Update()

Find the `case worktreeCreatedForNewSessionMsg:` handler. Add immediately after it:

```go
case reviewPRResolvedMsg:
    if msg.err != nil {
        h.reviewDialog.SetError(msg.err.Error())
        return h, nil
    }
    h.reviewDialog.SetResolved(msg.branch, msg.title, msg.isPR, msg.prNum)
    return h, nil
```

### Step 4: Add `resolvePRBranch` standalone function

Find the `finishWorktree` function (around line 9422). Add before it:

```go
// resolvePRBranch calls `gh pr view` to get the branch name and title for a PR number.
// Returns the branch name and PR title, or an error if gh is unavailable or PR not found.
func resolvePRBranch(repoDir, prNum string) (branch, title string, err error) {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "gh", "pr", "view", prNum,
        "--json", "headRefName,title",
        "--jq", ".headRefName+\"\\t\"+.title")
    cmd.Dir = repoDir
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", "", fmt.Errorf("gh pr view failed: %s", strings.TrimSpace(string(output)))
    }

    parts := strings.SplitN(strings.TrimSpace(string(output)), "\t", 2)
    if len(parts) == 0 || parts[0] == "" {
        return "", "", fmt.Errorf("could not determine branch for PR #%s", prNum)
    }
    branch = parts[0]
    if len(parts) == 2 {
        title = parts[1]
    }
    return branch, title, nil
}
```

> Note: this requires `"os/exec"`, `"context"`, `"time"` — check existing imports in home.go; they're almost certainly already there since the worktree code uses them. If not, add to the import block.

### Step 5: Add `createReviewSession` method

Add after `resolvePRBranch`:

```go
// createReviewSession fetches the branch, creates a worktree, starts a Claude session,
// and schedules the /pr-review prompt delivery. Returns a tea.Cmd.
func (h *Home) createReviewSession(
    repoDir, branch, prNum, sessionName, groupPath, initialPrompt string,
) tea.Cmd {
    return func() tea.Msg {
        // Step 1: Fetch the remote branch so it exists locally
        if err := git.FetchBranch(repoDir, branch); err != nil {
            return reviewSessionCreatedMsg{err: fmt.Errorf("fetch branch: %w", err)}
        }

        // Step 2: Generate worktree path using the session name as the label
        wtSettings := session.GetWorktreeSettings()
        worktreePath := git.GenerateWorktreePath(repoDir, sessionName, wtSettings.Location)

        // Step 3: Create the worktree (branch already exists locally from fetch)
        if err := git.CreateWorktree(repoDir, worktreePath, branch); err != nil {
            return reviewSessionCreatedMsg{err: fmt.Errorf("create worktree: %w", err)}
        }

        // Step 4: Check tmux availability
        if err := tmux.IsTmuxAvailable(); err != nil {
            return reviewSessionCreatedMsg{err: fmt.Errorf("tmux not available: %w", err)}
        }

        // Step 5: Create and start the session
        inst := session.NewInstanceWithGroupAndTool(sessionName, worktreePath, groupPath, "claude")
        inst.WorktreePath = worktreePath
        inst.WorktreeRepoRoot = repoDir
        inst.WorktreeBranch = branch

        if err := inst.Start(); err != nil {
            return reviewSessionCreatedMsg{err: fmt.Errorf("start session: %w", err)}
        }

        return reviewSessionCreatedMsg{
            instance:      inst,
            initialPrompt: initialPrompt,
        }
    }
}
```

### Step 6: Handle `reviewSessionCreatedMsg` in Update()

Find the `case reviewPRResolvedMsg:` handler just added. Add immediately after:

```go
case reviewSessionCreatedMsg:
    if msg.err != nil {
        h.setError(msg.err)
        return h, nil
    }
    inst := msg.instance
    h.instancesMu.Lock()
    h.instances = append(h.instances, inst)
    h.instanceByID[inst.ID] = inst
    h.instancesMu.Unlock()
    h.groupTree.AddSession(inst)
    h.rebuildFlatItems()
    h.search.SetItems(h.instances)
    h.forceSaveInstances()

    if msg.initialPrompt == "" {
        return h, nil
    }
    // Schedule prompt delivery after Claude has had time to start
    capturedInst := inst
    capturedPrompt := msg.initialPrompt
    return h, func() tea.Msg {
        time.Sleep(4 * time.Second)
        if err := capturedInst.SendText(capturedPrompt); err != nil {
            uiLog.Warn("review_prompt_send_failed",
                slog.String("id", capturedInst.ID),
                slog.String("err", err.Error()))
        }
        return reviewPromptSentMsg{}
    }
```

### Step 7: Handle `reviewPromptSentMsg` (no-op, needed to complete the Cmd chain)

Add after the `reviewSessionCreatedMsg` handler:

```go
case reviewPromptSentMsg:
    return h, nil
```

### Step 8: Build check

```bash
go build ./...
```

Fix any compilation errors (imports, field names, etc.).

### Step 9: Run all tests

```bash
go test ./internal/ui/... -run TestReviewDialog -v
go test ./internal/git/... -run TestFetchBranch -v
```

Expected: all `TestReviewDialog_*` and `TestFetchBranch_*` PASS. The two known pre-existing failures are expected.

### Step 10: Commit

```bash
git add internal/ui/home.go
git commit -m "feat(ui): implement review session creation with async PR resolution and prompt delivery"
```

---

## Task 7: Help overlay entry + smoke test

**Files:**
- Modify: `internal/ui/home.go` (help overlay key list)

### Step 1: Find the help overlay key definitions

```bash
grep -n "worktree\|\"W\"\|\"x\"\|\"n\"" internal/ui/help_overlay.go 2>/dev/null || \
grep -n "worktree\|\"W\"\|\"x\"\|\"n\"" internal/ui/home.go | grep -i "help\|key.*desc\|desc.*key" | head -20
```

### Step 2: Add the `v` key entry

Find where `W` is documented (something like `{"W", "worktree finish"}`). Add nearby:

```go
{"v", "review PR"},
```

Use the exact same struct/format as the surrounding entries.

### Step 3: Build and run all tests

```bash
go build ./...
go test ./internal/ui/... -run TestReviewDialog -v
go test ./internal/git/... -v
go test ./internal/session/... -v
```

Expected: `TestReviewDialog_*` PASS, `TestFetchBranch_InvalidBranch` PASS, no new failures.

### Step 4: Final commit

```bash
git add internal/ui/home.go internal/ui/help_overlay.go 2>/dev/null; git add -u
git commit -m "feat(ui): add 'v' key to help overlay for PR review"
```

---

## Acceptance Criteria

- [ ] `v` key is a no-op when no project/session is selected
- [ ] `v` on a session opens ReviewDialog pre-filled with project name
- [ ] `v` on a project group opens ReviewDialog pre-filled with group name
- [ ] Entering a PR number resolves to branch name + title via `gh pr view`
- [ ] If `gh` fails, inline error shown; input stays editable
- [ ] Entering a branch name skips resolution and goes to confirm step directly
- [ ] `esc` in confirm step returns to input step (not close)
- [ ] `esc` in input step closes dialog
- [ ] On confirm: worktree created at `.worktrees/review-pr-<N>` (or `review-<branch>`)
- [ ] Session appears in sidebar under project group
- [ ] `/pr-review <N>` (or branch) sent to Claude ~4s after session start
- [ ] Session can be cleaned up with `W` (worktree finish) like any other worktree session

---

## Known Pre-existing Test Failures (ignore)

These two tests were already failing before this work:
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`
