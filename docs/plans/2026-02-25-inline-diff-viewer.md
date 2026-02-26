# Inline Git Diff Viewer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a git diff viewer to Hangar — diffstat in the preview pane, full-screen scrollable overlay on `D`, open file in `$EDITOR` on `e`.

**Architecture:** Run `git diff` as a subprocess, parse via `go-diff`, render with lipgloss using existing oasis_lagoon_dark/dawn color variables. Two views share one parsed diff: a summary line in the preview pane and a full-screen overlay.

**Tech Stack:** Go, `github.com/sourcegraph/go-diff`, `github.com/charmbracelet/lipgloss`, Bubble Tea

---

### Task 1: Add go-diff dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/feature-sugges-t-features
go get github.com/sourcegraph/go-diff@latest
```

**Step 2: Verify it resolved**

```bash
grep "sourcegraph/go-diff" go.mod
```
Expected: a line like `github.com/sourcegraph/go-diff v0.7.0`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add go-diff dependency for inline diff viewer"
```

---

### Task 2: Add `FetchDiff` to the git package

**Files:**
- Create: `internal/git/diff.go`
- Test: `internal/git/diff_test.go`

**Step 1: Write the failing tests**

Create `internal/git/diff_test.go`:

```go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ghe.spotify.net/mnicholson/hangar/internal/git"
)

// makeRepo creates a temporary git repo for testing
func makeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	// Write and commit a base file
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial")
	return dir
}

func TestFetchDiff_CleanTree(t *testing.T) {
	dir := makeRepo(t)
	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for clean tree, got: %q", diff)
	}
}

func TestFetchDiff_WithChanges(t *testing.T) {
	dir := makeRepo(t)
	// Modify the committed file
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "world") {
		t.Errorf("expected diff to contain 'world', got: %q", diff)
	}
	if !strings.Contains(diff, "hello.txt") {
		t.Errorf("expected diff to mention hello.txt, got: %q", diff)
	}
}

func TestFetchDiff_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := git.FetchDiff(dir)
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestFetchDiff_WorktreeBranch(t *testing.T) {
	dir := makeRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// Create a feature branch with a new commit
	run("checkout", "-b", "feature/test")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "add new.txt")

	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain the new file from branch diff
	if !strings.Contains(diff, "new.txt") {
		t.Errorf("expected diff to contain new.txt, got: %q", diff)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/git/... -run TestFetchDiff -v
```
Expected: FAIL with "undefined: git.FetchDiff"

**Step 3: Implement `FetchDiff`**

Create `internal/git/diff.go`:

```go
package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotGitRepo is returned when the directory is not a git repository.
var ErrNotGitRepo = errors.New("not a git repository")

// FetchDiff returns the unified diff for the given directory.
//
// For worktree branches (non-main/master HEAD), it diffs the branch against
// its upstream or merge base: git diff <base>...HEAD
// For regular sessions (main/master or detached), it diffs working tree
// against HEAD: git diff HEAD
//
// Returns ("", nil) for a clean tree.
// Returns ("", ErrNotGitRepo) if dir is not a git repo.
func FetchDiff(dir string) (string, error) {
	if !IsGitRepo(dir) {
		return "", ErrNotGitRepo
	}

	// Determine the diff command based on branch state
	args := diffArgs(dir)

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		// git diff exits 1 when there are differences — that's normal
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return string(out), nil
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(out), nil
}

// diffArgs returns the git diff arguments appropriate for the repo state.
func diffArgs(dir string) []string {
	// Try to find a merge base with the default branch
	for _, base := range []string{"origin/main", "origin/master", "main", "master"} {
		cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", base)
		if err := cmd.Run(); err == nil {
			// base exists — use triple-dot diff (changes on this branch only)
			return []string{"diff", base + "...HEAD"}
		}
	}
	// No known base branch — fall back to working tree vs HEAD
	return []string{"diff", "HEAD"}
}

// DiffSummary parses a unified diff string and returns a short summary line.
// Returns "no changes" for an empty diff.
// Format: "N file(s), +X -Y"
func DiffSummary(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "no changes"
	}

	var files, additions, deletions int
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			files++
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			additions++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			deletions++
		}
	}

	noun := "file"
	if files != 1 {
		noun = "files"
	}
	return fmt.Sprintf("%d %s, +%d -%d", files, noun, additions, deletions)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/... -run TestFetchDiff -v
```
Expected: all 4 tests PASS

**Step 5: Commit**

```bash
git add internal/git/diff.go internal/git/diff_test.go
git commit -m "feat(git): add FetchDiff and DiffSummary for inline diff viewer"
```

---

### Task 3: Build `DiffView` struct (parse + summary)

**Files:**
- Create: `internal/ui/diff_view.go`
- Test: `internal/ui/diff_view_test.go`

**Step 1: Write failing tests for Parse and Summary**

Create `internal/ui/diff_view_test.go`:

```go
package ui

import (
	"strings"
	"testing"
)

const sampleDiff = `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 package main

+// added comment
 func main() {}
diff --git a/bar.go b/bar.go
index 111..222 100644
--- a/bar.go
+++ b/bar.go
@@ -1,2 +1,1 @@
 package main
-// removed line
`

func TestDiffView_ParseAndSummary(t *testing.T) {
	dv := NewDiffView()
	if err := dv.Parse(sampleDiff); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if dv.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", dv.FileCount())
	}
	summary := dv.Summary()
	if !strings.Contains(summary, "2 file") {
		t.Errorf("expected '2 file' in summary, got %q", summary)
	}
}

func TestDiffView_EmptyDiff(t *testing.T) {
	dv := NewDiffView()
	if err := dv.Parse(""); err != nil {
		t.Fatalf("Parse of empty string failed: %v", err)
	}
	if dv.Summary() != "no changes" {
		t.Errorf("expected 'no changes', got %q", dv.Summary())
	}
}

func TestDiffView_IsVisible(t *testing.T) {
	dv := NewDiffView()
	if dv.IsVisible() {
		t.Error("new DiffView should not be visible")
	}
	dv.Show()
	if !dv.IsVisible() {
		t.Error("DiffView should be visible after Show()")
	}
	dv.Hide()
	if dv.IsVisible() {
		t.Error("DiffView should not be visible after Hide()")
	}
}

func TestDiffView_FileUnderCursor(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.SetSize(120, 40)
	// At scrollOffset 0, first file header should be "foo.go"
	path, line := dv.FileUnderCursor()
	if !strings.Contains(path, "foo.go") {
		t.Errorf("expected foo.go under cursor at top, got %q", path)
	}
	if line < 1 {
		t.Errorf("expected line >= 1, got %d", line)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/... -run TestDiffView -v
```
Expected: FAIL with "undefined: NewDiffView"

**Step 3: Implement `DiffView` struct (parse + summary only for now)**

Create `internal/ui/diff_view.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
)

// DiffView renders a parsed git diff as a scrollable full-screen overlay.
// It follows the same Show/Hide/IsVisible/SetSize contract as other overlays
// (HelpOverlay, GlobalSearch, etc.) so it slots into home.go's existing
// overlay routing pattern with minimal changes.
type DiffView struct {
	visible      bool
	width        int
	height       int
	scrollOffset int

	// Parsed diff data
	files  []*diff.FileDiff
	rawDiff string

	// Rendered lines cache (rebuilt on Parse or SetSize)
	lines []renderedLine
}

// renderedLine holds a pre-rendered string and the source file path + first
// changed line number it belongs to (used by FileUnderCursor).
type renderedLine struct {
	text     string
	filePath string // empty for non-file-header lines
	line     int    // first changed line in this file (0 if unknown)
}

// NewDiffView creates a new DiffView.
func NewDiffView() *DiffView {
	return &DiffView{}
}

// Show makes the overlay visible and resets scroll to top.
func (dv *DiffView) Show() {
	dv.visible = true
	dv.scrollOffset = 0
}

// Hide hides the overlay.
func (dv *DiffView) Hide() {
	dv.visible = false
}

// IsVisible returns whether the overlay is currently shown.
func (dv *DiffView) IsVisible() bool {
	return dv.visible
}

// SetSize updates the terminal dimensions.
func (dv *DiffView) SetSize(width, height int) {
	dv.width = width
	dv.height = height
	dv.rebuildLines()
}

// Parse parses a raw unified diff string.
// An empty string is valid (represents a clean working tree).
func (dv *DiffView) Parse(raw string) error {
	dv.rawDiff = raw
	dv.files = nil
	dv.lines = nil

	if strings.TrimSpace(raw) == "" {
		return nil
	}

	files, err := diff.ParseMultiFileDiff([]byte(raw))
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}
	dv.files = files
	dv.rebuildLines()
	return nil
}

// FileCount returns the number of changed files.
func (dv *DiffView) FileCount() int {
	return len(dv.files)
}

// Summary returns a short human-readable summary of the diff.
// e.g. "3 files, +47 -12" or "no changes"
func (dv *DiffView) Summary() string {
	if len(dv.files) == 0 {
		return "no changes"
	}
	var additions, deletions int
	for _, f := range dv.files {
		for _, h := range f.Hunks {
			for _, line := range strings.Split(string(h.Body), "\n") {
				switch {
				case strings.HasPrefix(line, "+"):
					additions++
				case strings.HasPrefix(line, "-"):
					deletions++
				}
			}
		}
	}
	noun := "file"
	if len(dv.files) != 1 {
		noun = "files"
	}
	return fmt.Sprintf("%d %s, +%d -%d", len(dv.files), noun, additions, deletions)
}

// FileUnderCursor returns the file path and first changed line number for the
// file whose header is at or above the current scroll position.
// Returns ("", 0) when the diff is empty.
func (dv *DiffView) FileUnderCursor() (string, int) {
	if len(dv.lines) == 0 {
		return "", 0
	}
	// Walk backwards from scrollOffset to find the most recent file header
	idx := dv.scrollOffset
	if idx >= len(dv.lines) {
		idx = len(dv.lines) - 1
	}
	for i := idx; i >= 0; i-- {
		if dv.lines[i].filePath != "" {
			return dv.lines[i].filePath, dv.lines[i].line
		}
	}
	return "", 0
}

// rebuildLines (re)builds the rendered line cache from dv.files.
// Called after Parse and SetSize.
func (dv *DiffView) rebuildLines() {
	dv.lines = nil
	if len(dv.files) == 0 {
		return
	}

	for _, f := range dv.files {
		// Strip a/ b/ prefixes from file paths
		path := strings.TrimPrefix(f.NewName, "b/")
		if path == "" || path == "/dev/null" {
			path = strings.TrimPrefix(f.OrigName, "a/")
		}

		// First changed line in this file (from first hunk)
		firstLine := 0
		if len(f.Hunks) > 0 {
			firstLine = int(f.Hunks[0].NewStartLine)
		}

		// File header line
		header := renderFileHeader(f, path)
		dv.lines = append(dv.lines, renderedLine{
			text:     header,
			filePath: path,
			line:     firstLine,
		})

		for _, h := range f.Hunks {
			// Hunk header
			hunkHeader := renderHunkHeader(h)
			dv.lines = append(dv.lines, renderedLine{text: hunkHeader})

			// Hunk body lines
			for _, line := range strings.Split(string(h.Body), "\n") {
				dv.lines = append(dv.lines, renderedLine{text: renderDiffLine(line)})
			}
		}

		// Blank separator between files
		dv.lines = append(dv.lines, renderedLine{text: ""})
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/ui/... -run TestDiffView -v
```
Expected: all 4 tests PASS

**Step 5: Commit**

```bash
git add internal/ui/diff_view.go internal/ui/diff_view_test.go
git commit -m "feat(ui): add DiffView struct with parse, summary, and file-under-cursor"
```

---

### Task 4: Add rendering helpers to `DiffView`

**Files:**
- Modify: `internal/ui/diff_view.go`

These are the lipgloss rendering functions referenced in Task 3 (`renderFileHeader`, `renderHunkHeader`, `renderDiffLine`) plus the `View()` method. They use the existing `ColorXxx` variables from `styles.go`.

**Step 1: Add rendering helpers and `View()` to `diff_view.go`**

Append the following to `internal/ui/diff_view.go`:

```go
import (
	// add these to the existing import block:
	"github.com/charmbracelet/lipgloss"
	// ... existing imports
)

// View renders the full-screen overlay. Returns an empty string when not visible.
func (dv *DiffView) View() string {
	if !dv.visible {
		return ""
	}

	// Overlay box
	boxStyle := lipgloss.NewStyle().
		Width(dv.width).
		Height(dv.height).
		Background(ColorBg).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	footerStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)

	var b strings.Builder

	// Header bar
	title := dv.Summary()
	b.WriteString(headerStyle.Render("  Diff: "+title) + "\n")
	b.WriteString(strings.Repeat("─", dv.width-2) + "\n")

	// Content lines
	contentHeight := dv.height - 4 // header + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(dv.lines) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
		b.WriteString(dimStyle.Render("  nothing to diff") + "\n")
	} else {
		end := dv.scrollOffset + contentHeight
		if end > len(dv.lines) {
			end = len(dv.lines)
		}
		start := dv.scrollOffset
		if start > len(dv.lines) {
			start = len(dv.lines)
		}
		for _, rl := range dv.lines[start:end] {
			line := rl.text
			// Truncate to terminal width
			if dv.width > 4 && len(line) > dv.width-4 {
				line = line[:dv.width-4]
			}
			b.WriteString(line + "\n")
		}
	}

	// Footer
	b.WriteString(strings.Repeat("─", dv.width-2) + "\n")
	b.WriteString(footerStyle.Render("  j/k scroll  e open editor  q/esc close"))

	return boxStyle.Render(b.String())
}

// renderFileHeader renders a diff file header line with accent color.
func renderFileHeader(f *diff.FileDiff, displayPath string) string {
	style := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	return style.Render("  " + displayPath)
}

// renderHunkHeader renders a @@ hunk header line in comment/dim color.
func renderHunkHeader(h *diff.Hunk) string {
	style := lipgloss.NewStyle().Foreground(ColorComment)
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OrigStartLine, h.OrigLines, h.NewStartLine, h.NewLines)
	if h.Section != "" {
		header += " " + h.Section
	}
	return style.Render("  " + header)
}

// renderDiffLine renders a single diff body line with appropriate color.
func renderDiffLine(line string) string {
	if line == "" {
		return ""
	}
	switch line[0] {
	case '+':
		return lipgloss.NewStyle().Foreground(ColorGreen).Render(line)
	case '-':
		return lipgloss.NewStyle().Foreground(ColorRed).Render(line)
	case '\\':
		// "\ No newline at end of file"
		return lipgloss.NewStyle().Foreground(ColorComment).Italic(true).Render(line)
	default:
		return lipgloss.NewStyle().Foreground(ColorTextDim).Render(line)
	}
}
```

> **Note on imports:** Go requires a single `import` block. Merge the new imports (`lipgloss`) into the existing block at the top of `diff_view.go` — do not add a second `import` block.

**Step 2: Build to check for compile errors**

```bash
go build ./internal/ui/...
```
Expected: no errors

**Step 3: Run all DiffView tests**

```bash
go test ./internal/ui/... -run TestDiffView -v
```
Expected: all tests still PASS

**Step 4: Commit**

```bash
git add internal/ui/diff_view.go
git commit -m "feat(ui): add lipgloss rendering to DiffView (View, renderDiffLine, etc.)"
```

---

### Task 5: Add scroll and key handling to `DiffView`

**Files:**
- Modify: `internal/ui/diff_view.go`
- Modify: `internal/ui/diff_view_test.go`

**Step 1: Write failing scroll tests**

Add to `internal/ui/diff_view_test.go`:

```go
func TestDiffView_Scroll(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.SetSize(120, 10)

	initial := dv.scrollOffset
	dv.ScrollDown(3)
	if dv.scrollOffset != initial+3 {
		t.Errorf("expected scrollOffset %d, got %d", initial+3, dv.scrollOffset)
	}

	dv.ScrollUp(1)
	if dv.scrollOffset != initial+2 {
		t.Errorf("expected scrollOffset %d, got %d", initial+2, dv.scrollOffset)
	}

	// Cannot scroll above 0
	dv.ScrollUp(999)
	if dv.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0 after large scroll up, got %d", dv.scrollOffset)
	}
}

func TestDiffView_HandleKey_Close(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()

	handled, _ := dv.HandleKey("q")
	if !handled {
		t.Error("expected q to be handled")
	}
	if dv.IsVisible() {
		t.Error("expected DiffView to be hidden after q")
	}
}

func TestDiffView_HandleKey_Scroll(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()
	dv.SetSize(120, 10)

	before := dv.scrollOffset
	handled, _ := dv.HandleKey("j")
	if !handled {
		t.Error("expected j to be handled")
	}
	if dv.scrollOffset != before+1 {
		t.Errorf("expected scroll +1, got offset %d (was %d)", dv.scrollOffset, before)
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/ui/... -run "TestDiffView_Scroll|TestDiffView_HandleKey" -v
```
Expected: FAIL with "undefined: dv.ScrollDown"

**Step 3: Add scroll methods and `HandleKey` to `diff_view.go`**

Append to `internal/ui/diff_view.go`:

```go
import (
	// add to existing import block:
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
)

// ScrollDown scrolls down by n lines, clamped to end of content.
func (dv *DiffView) ScrollDown(n int) {
	max := len(dv.lines) - 1
	if max < 0 {
		max = 0
	}
	dv.scrollOffset += n
	if dv.scrollOffset > max {
		dv.scrollOffset = max
	}
}

// ScrollUp scrolls up by n lines, clamped to 0.
func (dv *DiffView) ScrollUp(n int) {
	dv.scrollOffset -= n
	if dv.scrollOffset < 0 {
		dv.scrollOffset = 0
	}
}

// HandleKey processes a key press when the overlay is visible.
// Returns (handled bool, cmd tea.Cmd).
func (dv *DiffView) HandleKey(key string) (bool, tea.Cmd) {
	if !dv.visible {
		return false, nil
	}

	pageSize := dv.height / 2
	if pageSize < 1 {
		pageSize = 5
	}

	switch key {
	case "q", "esc", "D":
		dv.Hide()
		return true, nil
	case "j", "down":
		dv.ScrollDown(1)
		return true, nil
	case "k", "up":
		dv.ScrollUp(1)
		return true, nil
	case "pgdown", "ctrl+d":
		dv.ScrollDown(pageSize)
		return true, nil
	case "pgup", "ctrl+u":
		dv.ScrollUp(pageSize)
		return true, nil
	case "e":
		path, line := dv.FileUnderCursor()
		if path != "" {
			dv.Hide()
			return true, openInEditor(path, line)
		}
		return true, nil
	}
	return false, nil
}

// openInEditor returns a tea.Cmd that opens the given file at the given line
// in $EDITOR (falls back to "vi").
func openInEditor(path string, line int) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	// Most editors accept +N to jump to line N
	var args []string
	if line > 0 {
		args = []string{fmt.Sprintf("+%d", line), path}
	} else {
		args = []string{path}
	}
	return tea.ExecProcess(exec.Command(editor, args...), func(err error) tea.Msg {
		return nil // errors are silently ignored; editor is best-effort
	})
}
```

> **Note on imports:** Merge `tea`, `os`, `os/exec` into the existing import block — do not add a second block.

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run "TestDiffView" -v
```
Expected: all tests PASS

**Step 5: Commit**

```bash
git add internal/ui/diff_view.go internal/ui/diff_view_test.go
git commit -m "feat(ui): add scroll, HandleKey, and open-in-editor to DiffView"
```

---

### Task 6: Add diffstat line to the preview pane

**Files:**
- Modify: `internal/ui/preview.go`
- Modify: `internal/ui/home.go` (add `diffSummary string` field and update on focus change)

The preview pane (`Preview.View()`) currently receives a pre-rendered `content` string. The cleanest approach is to add a `DiffSummary` field and render it as an extra line.

**Step 1: Add `DiffSummary` to `Preview`**

In `internal/ui/preview.go`:

1. Add field to struct:
```go
type Preview struct {
	content     string
	title       string
	width       int
	height      int
	DiffSummary string // "3 files, +47 -12" or "" to hide
}
```

2. In `View()`, add the diffstat line after the separator and before content. Find the line `b.WriteString("\n\n")` (after the `─` separator) and replace it with:

```go
	b.WriteString("\n")
	if p.DiffSummary != "" {
		dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
		b.WriteString(dimStyle.Render("  diff  " + p.DiffSummary))
		b.WriteString("\n")
	}
	b.WriteString("\n")
```

**Step 2: Build to check for compile errors**

```bash
go build ./internal/ui/...
```
Expected: no errors

**Step 3: Update `home.go` to set `DiffSummary` on focus change**

In `internal/ui/home.go`:

1. Add field to `homeModel`:
```go
diffView    *DiffView
diffRawLast string // last fetched raw diff (to skip re-parse on same content)
```

2. In the init section (where other dialogs are created), add:
```go
h.diffView = NewDiffView()
```

3. Find where the preview content is set (search for `h.preview.SetContent`). After that call, add:
```go
// Update diffstat in preview pane
if inst != nil {
    go func(dir string) {
        raw, err := git.FetchDiff(dir)
        if err != nil {
            h.preview.DiffSummary = ""
            return
        }
        _ = h.diffView.Parse(raw)
        h.preview.DiffSummary = h.diffView.Summary()
    }(effectiveDir(inst)) // effectiveDir returns WorktreePath or ProjectPath
}
```

> `effectiveDir` is a small helper — see Step 4.

4. Add helper (can go in `home.go` or a new small file):
```go
// effectiveDir returns the working directory to use for git operations on inst.
// For worktree sessions it returns WorktreePath; otherwise ProjectPath.
func effectiveDir(inst *session.Instance) string {
	if inst.WorktreePath != "" {
		return inst.WorktreePath
	}
	return inst.ProjectPath
}
```

**Step 4: Build and run UI tests**

```bash
go build ./...
go test ./internal/ui/... -v
```
Expected: build succeeds, existing tests pass

**Step 5: Commit**

```bash
git add internal/ui/preview.go internal/ui/home.go
git commit -m "feat(ui): show diffstat summary in preview pane"
```

---

### Task 7: Wire `DiffView` overlay into `home.go`

This task adds the `D` key binding and wires the overlay into the 7 standard Bubble Tea integration points.

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add `DiffView` to the 7 integration points**

Open `internal/ui/home.go` and make the following changes. Search for patterns from the existing `helpOverlay` wiring — each change mirrors it exactly.

**1. Key routing** — find the block starting with `// Handle overlays first` (~line 3200):
```go
if h.diffView.IsVisible() {
    if handled, cmd := h.diffView.HandleKey(msg.String()); handled {
        return h, cmd
    }
}
```
Add this *before* the `h.helpOverlay.IsVisible()` check.

**2. `D` trigger key** — in the session list key handler (the `default:` section or near `case "o":`), add:
```go
case "D":
    inst := h.selectedSession()
    if inst != nil {
        dir := effectiveDir(inst)
        if git.IsGitRepo(dir) {
            h.diffView.Show()
        }
    }
```

**3. Mouse guard** — find the long `if h.setupWizard.IsVisible() || ...` guard (~line 3537), add `h.diffView.IsVisible() ||` to the chain.

**4. `SetSize`** — find where `h.helpOverlay.SetSize(...)` is called and add alongside it:
```go
h.diffView.SetSize(h.width, h.height)
```

**5. `View` check** — find the View rendering chain (`if h.helpOverlay.IsVisible() { return h.helpOverlay.View() }`), add:
```go
if h.diffView.IsVisible() {
    return h.diffView.View()
}
```

**Step 2: Add import for git package** (if not already present)

In `home.go`'s import block, ensure:
```go
"ghe.spotify.net/mnicholson/hangar/internal/git"
```

**Step 3: Build**

```bash
go build ./...
```
Expected: no errors

**Step 4: Run all tests**

```bash
go test ./...
```
Expected: all tests pass (the two pre-existing known failures in `internal/ui/` are acceptable)

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): wire DiffView overlay into home.go (D key, SetSize, View, guards)"
```

---

### Task 8: Handle the async diff fetch with debounce

The goroutine added in Task 6 fires a raw goroutine, which is not idiomatic Bubble Tea (state mutations from goroutines cause races). Replace it with a proper `tea.Cmd` + `tea.Msg` pattern.

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add a `diffFetchedMsg` type**

Near the other message type definitions in `home.go` (search for `type sendTextResultMsg`):

```go
// diffFetchedMsg is sent when a git diff has been fetched for the focused session.
type diffFetchedMsg struct {
	raw string
	err error
}
```

**Step 2: Add a `fetchDiffCmd` function**

```go
func fetchDiffCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		raw, err := git.FetchDiff(dir)
		return diffFetchedMsg{raw: raw, err: err}
	}
}
```

**Step 3: Replace the goroutine with `tea.Cmd`**

Remove the `go func(dir string) { ... }(effectiveDir(inst))` block added in Task 6.

Replace the preview update logic with:
```go
if inst != nil {
    if git.IsGitRepo(effectiveDir(inst)) {
        return h, fetchDiffCmd(effectiveDir(inst))
    }
    h.preview.DiffSummary = ""
    _ = h.diffView.Parse("")
}
```

**Step 4: Handle `diffFetchedMsg` in `Update()`**

Find the large `switch msg.(type)` in `Update()` and add a case:

```go
case diffFetchedMsg:
    if msg.err == nil {
        _ = h.diffView.Parse(msg.raw)
        h.preview.DiffSummary = h.diffView.Summary()
    } else {
        h.preview.DiffSummary = "diff unavailable"
    }
    return h, nil
```

**Step 5: Build and test**

```bash
go build ./...
go test ./...
```
Expected: no errors, all tests pass

**Step 6: Commit**

```bash
git add internal/ui/home.go
git commit -m "refactor(ui): use tea.Cmd/Msg for async diff fetch (replace raw goroutine)"
```

---

### Task 9: Update help overlay with `D` key binding

**Files:**
- Modify: `internal/ui/help.go`

**Step 1: Find the keybinding list in `help.go`**

Search for an existing entry like `"o"` or `"G"` to find the section structure. Add an entry for `D`:

```go
{"D", "diff", "show git diff for focused session"},
```

Place it near `o` (open PR) since both are session-level actions.

**Step 2: Build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/ui/help.go
git commit -m "docs(ui): add D key binding to help overlay"
```

---

### Task 10: Final validation

**Step 1: Run the full test suite**

```bash
go test ./...
```
Expected: all tests pass (ignore the two known pre-existing failures in `internal/ui/`)

**Step 2: Build the binary**

```bash
go build -o /tmp/hangar-diff-test ./cmd/hangar
```
Expected: binary produced with no errors

**Step 3: Manual smoke test** (optional but recommended)

```bash
# Navigate to any git repo, launch hangar, select a session with changes, press D
/tmp/hangar-diff-test
```

Verify:
- Preview pane shows "diff  N files, +X -Y" (or "no changes")
- `D` opens the overlay
- `j`/`k` scroll the diff
- `e` closes the overlay and opens `$EDITOR` on the correct file
- `q`/`esc` closes the overlay
- Non-git sessions: `D` does nothing, no diffstat in preview

**Step 4: Final commit (if any cleanup needed)**

```bash
git add -p  # stage only intentional changes
git commit -m "fix: [describe any final cleanup]"
```
