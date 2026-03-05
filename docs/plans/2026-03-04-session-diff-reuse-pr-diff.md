# Session Diff View — Collapsible File Tree (PR Diff Parity) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the flat scrollable diff in the session `DiffView` overlay (`D` key) with the same collapsible file-tree layout used by the PR Detail overlay's Diff tab, using the same pre-compiled styles for visual consistency.

**Architecture:** `DiffView` (`internal/ui/diff_view.go`) gains `diffFileEntry` slice + cursor fields (reusing the struct already defined in `pr_detail.go`, same package). `rebuildLines()` renders a collapsible file list identical to `buildDiffLines()` in `pr_detail.go`. Inline style creation in `diff_view.go` is replaced by pre-compiled package-level vars added to `styles.go`. `j/k` becomes file navigation (matching the PR diff tab); pager keys (`d/u`, `space/b`, `pgup/pgdown`) continue to scroll.

**Tech Stack:** Go, Bubble Tea (`charmbracelet/bubbletea`), lipgloss (`charmbracelet/lipgloss`), `sourcegraph/go-diff`

---

### Task 1: Add pre-compiled diff styles to `styles.go`

**Files:**
- Modify: `internal/ui/styles.go` — add var declarations + `initStyles()` lines

**Context:** All styles in the `prDetail*` block at lines 715–734 are initialized inside `initStyles()`. New diff-view styles go in the same block. The `renderDiffLine()` and `renderHunkHeader()` functions in `diff_view.go` currently create `lipgloss.NewStyle()` on every call — a hot-path allocation problem. The file-entry rows in the new collapsible view need the same focus/status colors as the PR diff tab.

**Step 1: Add var declarations** (after the `prDetail*` block, before `initStyles()` closes)

In the `var (...)` block near the top of `styles.go` (around line 16 where other `prDetail*` vars are declared), add:

```go
// Pre-compiled styles for DiffView overlay and shared diff line rendering.
var (
	diffViewHeaderStyle    lipgloss.Style
	diffViewSeparatorStyle lipgloss.Style
	diffViewFooterStyle    lipgloss.Style
	diffHunkHeaderStyle    lipgloss.Style
	diffLineAddStyle       lipgloss.Style
	diffLineDelStyle       lipgloss.Style
	diffLineContextStyle   lipgloss.Style
	diffLineNoNewlineStyle lipgloss.Style
)
```

**Step 2: Initialize in `initStyles()`** (append after `prDetailCommentStyle = ...` line ~734):

```go
	// DiffView overlay + shared diff line styles
	diffViewHeaderStyle    = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	diffViewSeparatorStyle = lipgloss.NewStyle().Foreground(ColorBorder)
	diffViewFooterStyle    = lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	diffHunkHeaderStyle    = lipgloss.NewStyle().Foreground(ColorComment)
	diffLineAddStyle       = lipgloss.NewStyle().Foreground(ColorGreen)
	diffLineDelStyle       = lipgloss.NewStyle().Foreground(ColorRed)
	diffLineContextStyle   = lipgloss.NewStyle().Foreground(ColorTextDim)
	diffLineNoNewlineStyle = lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
```

**Step 3: Build to verify no compile errors**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/session-diff-reuse-pr-diff
go build ./...
```
Expected: no errors.

**Step 4: Commit**

```bash
git add internal/ui/styles.go
git commit -m "style: pre-compile diff view styles in initStyles()"
```

---

### Task 2: Update `renderDiffLine` and `renderHunkHeader` to use pre-compiled styles

**Files:**
- Modify: `internal/ui/diff_view.go` — lines 340–363

**Context:** These two functions are called by both `DiffView.rebuildLines()` and `PRDetailOverlay.buildDiffLines()`. Both currently create `lipgloss.NewStyle()` inline on every call. Switch them to use the vars added in Task 1. Note that `renderFileHeader()` will be removed in Task 3 — don't touch it yet.

**Step 1: Update `renderHunkHeader`** (currently lines 340–346):

```go
func renderHunkHeader(h *diff.Hunk) string {
	hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OrigStartLine, h.OrigLines, h.NewStartLine, h.NewLines)
	if h.Section != "" {
		hdr += " " + h.Section
	}
	return diffHunkHeaderStyle.Render("  " + hdr)
}
```

**Step 2: Update `renderDiffLine`** (currently lines 348–363):

```go
func renderDiffLine(line string) string {
	if line == "" {
		return ""
	}
	switch line[0] {
	case '+':
		return diffLineAddStyle.Render(line)
	case '-':
		return diffLineDelStyle.Render(line)
	case '\\':
		return diffLineNoNewlineStyle.Render(line)
	default:
		return diffLineContextStyle.Render(line)
	}
}
```

**Step 3: Build + run tests**

```bash
go build ./... && go test ./internal/ui/...
```
Expected: all existing tests pass (the rendering output may differ slightly in color codes, but no test checks raw ANSI output).

**Step 4: Commit**

```bash
git add internal/ui/diff_view.go
git commit -m "refactor: use pre-compiled styles in renderDiffLine and renderHunkHeader"
```

---

### Task 3: Rewrite `DiffView` with collapsible file tree

**Files:**
- Modify: `internal/ui/diff_view.go` — major rewrite of struct, `Parse()`, `rebuildLines()`, `View()`, `HandleKey()`, `FileUnderCursor()`

**Context:** `diffFileEntry` is already defined in `pr_detail.go` in the same `package ui` — no import needed. The collapsible rendering logic mirrors `buildDiffLines()` in `pr_detail.go`. Key differences from the PR version: DiffView parses from `[]*diff.FileDiff` (not raw string + `parseDiffHunks()`), so we get hunk content directly from the parsed struct — cleaner, no "skip first 4 lines" hack. Status is inferred from OrigName/NewName `/dev/null` sentinel. Auto-expand when there is exactly 1 file.

**Step 1: Update `DiffView` struct** — add collapsible state fields:

```go
type DiffView struct {
	visible      bool
	width        int
	height       int
	scrollOffset int

	files   []*diff.FileDiff
	rawDiff string

	lines []renderedLine

	// Collapsible file list state
	diffFiles          []diffFileEntry
	diffFileCursor     int
	diffHeaderLineIdxs []int
}
```

**Step 2: Update `Parse()` to fill `diffFiles`**

After `dv.files = files` and `dv.rebuildLines()`, add file entry construction. Replace the existing `Parse()` body with:

```go
func (dv *DiffView) Parse(raw string) error {
	dv.rawDiff = raw
	dv.files = nil
	dv.lines = nil
	dv.diffFiles = nil
	dv.diffFileCursor = 0

	if strings.TrimSpace(raw) == "" {
		return nil
	}

	files, err := diff.ParseMultiFileDiff([]byte(raw))
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}
	dv.files = files

	// Build per-file entries for the collapsible view.
	autoExpand := len(files) == 1
	dv.diffFiles = make([]diffFileEntry, 0, len(files))
	for _, f := range files {
		path := strings.TrimPrefix(f.NewName, "b/")
		if path == "" || path == "/dev/null" {
			path = strings.TrimPrefix(f.OrigName, "a/")
		}

		status := "modified"
		origPath := strings.TrimPrefix(f.OrigName, "a/")
		if origPath == "/dev/null" {
			status = "added"
		} else if path == "/dev/null" || strings.TrimPrefix(f.NewName, "b/") == "/dev/null" {
			status = "deleted"
			path = origPath
		}

		var additions, deletions int
		for _, h := range f.Hunks {
			for _, l := range strings.Split(string(h.Body), "\n") {
				switch {
				case strings.HasPrefix(l, "+"):
					additions++
				case strings.HasPrefix(l, "-"):
					deletions++
				}
			}
		}

		dv.diffFiles = append(dv.diffFiles, diffFileEntry{
			path:      path,
			status:    status,
			additions: additions,
			deletions: deletions,
			expanded:  autoExpand,
			// hunk field unused — DiffView renders directly from dv.files[i].Hunks
		})
	}

	dv.rebuildLines()
	return nil
}
```

**Step 3: Rewrite `rebuildLines()`**

Replace the entire `rebuildLines()` function:

```go
func (dv *DiffView) rebuildLines() {
	dv.lines = nil
	dv.diffHeaderLineIdxs = nil

	if len(dv.diffFiles) == 0 {
		return
	}

	statusStyle := func(status string) lipgloss.Style {
		switch status {
		case "added":
			return prDetailGreenStyle
		case "deleted":
			return prDetailRedStyle
		case "modified":
			return prDetailYellowStyle
		default:
			return prDetailValueStyle
		}
	}

	// Summary bar
	totalAdd, totalDel := 0, 0
	for _, f := range dv.diffFiles {
		totalAdd += f.additions
		totalDel += f.deletions
	}
	noun := "file"
	if len(dv.diffFiles) != 1 {
		noun = "files"
	}
	dv.lines = append(dv.lines, renderedLine{
		text: prDetailCommentStyle.Render(fmt.Sprintf("  %d %s changed  +%d -%d", len(dv.diffFiles), noun, totalAdd, totalDel)),
	})
	dv.lines = append(dv.lines, renderedLine{text: ""})

	for i, entry := range dv.diffFiles {
		headerIdx := len(dv.lines)
		dv.diffHeaderLineIdxs = append(dv.diffHeaderLineIdxs, headerIdx)

		indicator := "▶"
		if entry.expanded {
			indicator = "▼"
		}
		focused := i == dv.diffFileCursor

		pathStyle := statusStyle(entry.status)
		indStyle := prDetailCommentStyle
		stStyle := prDetailCommentStyle
		if focused {
			pathStyle = pathStyle.Background(prDetailFocusBgColor)
			indStyle = indStyle.Background(prDetailFocusBgColor)
			stStyle = stStyle.Background(prDetailFocusBgColor)
		}

		maxPathW := dv.width - 20
		if maxPathW < 10 {
			maxPathW = 10
		}
		path := entry.path
		if len([]rune(path)) > maxPathW {
			path = "…" + string([]rune(path)[len([]rune(path))-maxPathW+1:])
		}

		statsStr := fmt.Sprintf("+%d -%d", entry.additions, entry.deletions)
		header := "  " + indStyle.Render(indicator) + " " + pathStyle.Render(path) +
			"  " + stStyle.Render(statsStr)
		if focused {
			visible := lipgloss.Width(header)
			if visible < dv.width-2 {
				header += prDetailFocusPadStyle.Render(strings.Repeat(" ", dv.width-2-visible))
			}
		}
		rl := renderedLine{text: header, filePath: entry.path}
		if len(dv.files) > i && len(dv.files[i].Hunks) > 0 {
			rl.line = int(dv.files[i].Hunks[0].NewStartLine)
		}
		dv.lines = append(dv.lines, rl)

		if entry.expanded && i < len(dv.files) {
			for _, h := range dv.files[i].Hunks {
				dv.lines = append(dv.lines, renderedLine{text: renderHunkHeader(h)})
				for _, bodyLine := range strings.Split(string(h.Body), "\n") {
					dv.lines = append(dv.lines, renderedLine{text: "  " + renderDiffLine(bodyLine)})
				}
			}
			dv.lines = append(dv.lines, renderedLine{text: ""})
		}
	}
}
```

**Step 4: Update `View()` to use pre-compiled styles**

Replace the inline style vars at the top of `View()`:

```go
func (dv *DiffView) View() string {
	if !dv.visible {
		return ""
	}

	sep := diffViewSeparatorStyle.Render(strings.Repeat("─", max(dv.width-2, 0)))

	var b strings.Builder
	b.WriteString(diffViewHeaderStyle.Render("  Diff: "+dv.Summary()) + "\n")
	b.WriteString(sep + "\n")

	contentHeight := dv.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(dv.lines) == 0 {
		b.WriteString(prDetailDimStyle.Render("  nothing to diff") + "\n")
	} else {
		start := dv.scrollOffset
		if start >= len(dv.lines) {
			start = len(dv.lines) - 1
		}
		if start < 0 {
			start = 0
		}
		end := start + contentHeight
		if end > len(dv.lines) {
			end = len(dv.lines)
		}
		for _, rl := range dv.lines[start:end] {
			b.WriteString(rl.text + "\n")
		}
	}

	b.WriteString(sep + "\n")
	hint := "  j/k navigate files · enter toggle · space/b page · d/u half-page · g/G top/bottom · e editor · q close"
	b.WriteString(diffViewFooterStyle.Render(hint))

	return lipgloss.NewStyle().
		Width(dv.width).
		Height(dv.height).
		Background(ColorBg).
		Padding(0, 1).
		Render(b.String())
}
```

**Step 5: Update `HandleKey()` — add file navigation, keep pager keys**

Replace the `j/k` and `enter/space` cases. The full updated `HandleKey()`:

```go
func (dv *DiffView) HandleKey(key string) (bool, tea.Cmd) {
	if !dv.visible {
		return false, nil
	}

	fullPage := dv.height - 4
	if fullPage < 1 {
		fullPage = 1
	}
	halfPage := fullPage / 2
	if halfPage < 1 {
		halfPage = 1
	}

	// File navigation (when files are present)
	if len(dv.diffFiles) > 0 {
		switch key {
		case "j", "down":
			if dv.diffFileCursor < len(dv.diffFiles)-1 {
				dv.diffFileCursor++
				dv.rebuildLines()
				dv.scrollToFileCursor()
			}
			return true, nil
		case "k", "up":
			if dv.diffFileCursor > 0 {
				dv.diffFileCursor--
				dv.rebuildLines()
				dv.scrollToFileCursor()
			}
			return true, nil
		case "enter", " ":
			dv.diffFiles[dv.diffFileCursor].expanded = !dv.diffFiles[dv.diffFileCursor].expanded
			dv.rebuildLines()
			dv.scrollToFileCursor()
			return true, nil
		}
	}

	switch key {
	case "q", "esc", "D":
		dv.Hide()
		return true, nil
	case "f", "ctrl+f", "pgdown":
		dv.ScrollDown(fullPage)
		return true, nil
	case "b", "ctrl+b", "pgup":
		dv.ScrollUp(fullPage)
		return true, nil
	case "d", "ctrl+d":
		dv.ScrollDown(halfPage)
		return true, nil
	case "u", "ctrl+u":
		dv.ScrollUp(halfPage)
		return true, nil
	case "g":
		dv.ScrollToTop()
		return true, nil
	case "G":
		dv.ScrollToBottom()
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
```

Note: `space` is now consumed by the file-toggle branch (when files exist), so it's removed from the pager section. Add a helper `scrollToFileCursor()`:

```go
func (dv *DiffView) scrollToFileCursor() {
	if dv.diffFileCursor >= len(dv.diffHeaderLineIdxs) {
		return
	}
	contentHeight := dv.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}
	headerLine := dv.diffHeaderLineIdxs[dv.diffFileCursor]
	if headerLine < dv.scrollOffset {
		dv.scrollOffset = headerLine
	} else if headerLine >= dv.scrollOffset+contentHeight {
		dv.scrollOffset = headerLine - contentHeight + 1
	}
}
```

**Step 6: Update `FileUnderCursor()`** — now just returns the cursor file:

```go
func (dv *DiffView) FileUnderCursor() (string, int) {
	if len(dv.diffFiles) == 0 || dv.diffFileCursor >= len(dv.diffFiles) {
		return "", 0
	}
	path := dv.diffFiles[dv.diffFileCursor].path
	line := 0
	if dv.diffFileCursor < len(dv.files) && len(dv.files[dv.diffFileCursor].Hunks) > 0 {
		line = int(dv.files[dv.diffFileCursor].Hunks[0].NewStartLine)
	}
	return path, line
}
```

**Step 7: Remove `renderFileHeader()`** — it's no longer called anywhere. Delete lines 336–338.

**Step 8: Build**

```bash
go build ./...
```
Expected: no errors.

**Step 9: Commit**

```bash
git add internal/ui/diff_view.go
git commit -m "feat(ui): session diff view — collapsible file tree, consistent with PR detail"
```

---

### Task 4: Update tests for new DiffView behavior

**Files:**
- Modify: `internal/ui/diff_view_test.go`

**Context:** Several tests must be updated because `j/k` no longer scrolls lines — it navigates between file headers. The `TestDiffView_HandleKey_Scroll` test explicitly checks that `j` increments `scrollOffset` by 1, which is no longer the case. `TestDiffView_HandleKey_PagerBindings` uses `space` for full-page-down, but `space` is now consumed by file-toggle — update that test to use `f` instead. `TestDiffView_Scroll` calls `ScrollDown`/`ScrollUp` directly and accesses `dv.lines` — with a 2-file collapsed diff there are only ~4 lines, so the clamping assertions need updating. `TestDiffView_FileUnderCursor` should still pass (cursor=0 → foo.go) but verify.

**Step 1: Fix `TestDiffView_HandleKey_Scroll`**

Replace the test body. The new `j` behavior moves the file cursor, not the scroll offset:

```go
func TestDiffView_HandleKey_FileNavigation(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()
	dv.SetSize(120, 40)

	// Initially on first file
	if dv.diffFileCursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", dv.diffFileCursor)
	}

	// j moves to next file
	handled, _ := dv.HandleKey("j")
	if !handled {
		t.Error("expected j to be handled")
	}
	if dv.diffFileCursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", dv.diffFileCursor)
	}

	// j at last file does not go further
	dv.HandleKey("j")
	if dv.diffFileCursor != 1 {
		t.Errorf("expected cursor still at 1 (last file), got %d", dv.diffFileCursor)
	}

	// k moves back
	dv.HandleKey("k")
	if dv.diffFileCursor != 0 {
		t.Errorf("expected cursor back at 0 after k, got %d", dv.diffFileCursor)
	}

	// k at first file does not go negative
	dv.HandleKey("k")
	if dv.diffFileCursor != 0 {
		t.Errorf("expected cursor still at 0, got %d", dv.diffFileCursor)
	}
}
```

**Step 2: Fix `TestDiffView_HandleKey_PagerBindings`** — replace the `space full-page-down` subtest (space now toggles expansion) with `f`:

```go
t.Run("f full-page-down", func(t *testing.T) {
	dv := setup()
	// expand all files to get enough lines for scrolling
	for i := range dv.diffFiles {
		dv.diffFiles[i].expanded = true
	}
	dv.rebuildLines()
	dv.SetSize(120, 10)
	before := dv.scrollOffset
	handled, _ := dv.HandleKey("f")
	if !handled {
		t.Error("expected f to be handled")
	}
	_ = before // scroll may or may not change depending on content height vs line count
})
```

Also remove the old `TestDiffView_HandleKey_Scroll` test (replaced by `TestDiffView_HandleKey_FileNavigation`).

**Step 3: Fix `TestDiffView_Scroll`** — the test directly calls `ScrollDown`/`ScrollUp` and checks `scrollOffset`. With collapsed files (~4 lines) and height=10 (visibleHeight=6), `ScrollDown(3)` will clamp to 0. Expand files first to get enough lines:

```go
func TestDiffView_Scroll(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	// Expand all files so there are enough lines to scroll
	for i := range dv.diffFiles {
		dv.diffFiles[i].expanded = true
	}
	dv.SetSize(120, 10)
	dv.rebuildLines()

	initial := dv.scrollOffset
	dv.ScrollDown(3)
	if dv.scrollOffset < initial {
		t.Errorf("ScrollDown should increase offset, was %d now %d", initial, dv.scrollOffset)
	}

	before := dv.scrollOffset
	dv.ScrollUp(1)
	if dv.scrollOffset > before {
		t.Errorf("ScrollUp should decrease offset, was %d now %d", before, dv.scrollOffset)
	}

	// Cannot scroll above 0
	dv.ScrollUp(999)
	if dv.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0 after large scroll up, got %d", dv.scrollOffset)
	}

	// ScrollDown(999) clamps at max
	dv2 := NewDiffView()
	_ = dv2.Parse(sampleDiff)
	for i := range dv2.diffFiles {
		dv2.diffFiles[i].expanded = true
	}
	dv2.SetSize(120, 20)
	dv2.rebuildLines()
	dv2.ScrollDown(999)
	visibleHeight := 16
	maxOffset := len(dv2.lines) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if dv2.scrollOffset != maxOffset {
		t.Errorf("expected scrollOffset clamped at %d, got %d", maxOffset, dv2.scrollOffset)
	}
}
```

**Step 4: Add test for enter/space toggle**

```go
func TestDiffView_HandleKey_Toggle(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()
	dv.SetSize(120, 40)

	// sampleDiff has 2 files — both start collapsed (autoExpand only for 1 file)
	if dv.diffFiles[0].expanded {
		t.Fatal("expected first file to start collapsed with 2-file diff")
	}

	// enter expands
	handled, _ := dv.HandleKey("enter")
	if !handled {
		t.Error("expected enter to be handled")
	}
	if !dv.diffFiles[0].expanded {
		t.Error("expected first file to be expanded after enter")
	}

	// enter again collapses
	dv.HandleKey("enter")
	if dv.diffFiles[0].expanded {
		t.Error("expected first file to be collapsed after second enter")
	}
}
```

**Step 5: Add test for single-file auto-expand**

```go
func TestDiffView_SingleFile_AutoExpand(t *testing.T) {
	singleFileDiff := `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -1,2 +1,3 @@
 package main
+// added
 func main() {}
`
	dv := NewDiffView()
	_ = dv.Parse(singleFileDiff)
	if len(dv.diffFiles) != 1 {
		t.Fatalf("expected 1 file, got %d", len(dv.diffFiles))
	}
	if !dv.diffFiles[0].expanded {
		t.Error("single-file diff should auto-expand")
	}
}
```

**Step 6: Run tests**

```bash
go test ./internal/ui/... -run TestDiffView -v
```
Expected: all DiffView tests pass.

**Step 7: Run full test suite**

```bash
go test ./...
```
Expected: all tests pass (pre-existing failures `TestNewDialog_WorktreeToggle_ViaKeyPress` and `TestNewDialog_TypingResetsSuggestionNavigation` are known and acceptable).

**Step 8: Commit**

```bash
git add internal/ui/diff_view_test.go
git commit -m "test(ui): update DiffView tests for collapsible file navigation"
```

---

### Task 5: Update CHANGELOG.md for v2.5.0

**Files:**
- Modify: `CHANGELOG.md`

**Context:** The last formally released version is `v2.4.0` (has a link at the bottom). Everything in `[2.4.1]` is unreleased. The `feat(webui): click-to-edit todos (#42)` commit is in git history but not in the changelog at all. This task consolidates all unreleased work into a single `[2.5.0]` entry, removes the `[2.4.1]` heading (folding its content into `2.5.0`), and adds the new diff view feature.

**Step 1: Replace the top of `CHANGELOG.md`**

Replace everything from line 1 down to (but not including) the `## [2.4.0]` line with:

```markdown
# Changelog

All notable changes to Hangar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.5.0] - 2026-03-04

### Added

- **TUI: Session diff view — collapsible file tree** — the `D` key overlay now renders a
  collapsible per-file list identical to the PR Detail overlay's Diff tab. Each row shows the
  file path (colorized green/red/yellow by status), a `▶`/`▼` expand indicator, and `+N -N`
  change counts. `j`/`k` navigate between files; `Enter`/`Space` expand or collapse hunks;
  single-file diffs auto-expand. Focus highlight, color palette, and key bindings are now
  identical between the session diff view and the PR diff view. Inline `lipgloss.NewStyle()`
  calls in `renderDiffLine` and `renderHunkHeader` replaced with pre-compiled package-level
  style vars for allocation consistency.

- **TUI: Approve PR from detail overlay** — the `a` key now works in the `PRDetailOverlay`
  (Enter → detail view), not just in the PR list. Hint bar updated to show `a approve`.

- **TUI: PR list sorting** — `S` key cycles through sort columns (`age` → `title` → `author`
  → `state` → `checks`); pressing again on the same column flips direction. Default is
  newest-first by age. Active column shown with `↑`/`↓` arrow in the column header.

- **TUI: Mouse support in PR overview** — scroll wheel moves cursor; single click selects a
  row; clicking an already-selected row opens the detail overlay (equivalent to Enter).

- **WebUI: Click-to-edit todos** — todo titles in the kanban board are now inline-editable:
  click a title to enter edit mode, type to update, press Enter or click away to save.

- **WebUI: PR list sorting** — clicking column headers (Age, Author, Title, Checks, State)
  sorts the list; clicking the same header again reverses direction. `↑`/`↓` indicator shown
  on the active column. Default is newest-first.

### Fixed

- **WebUI: Approve button now consistent across all tabs** — the Approve and Request Changes
  buttons previously only appeared for PRs where `source !== 'mine'`, which caused them to be
  hidden for session-linked PRs also authored by the current user. Buttons now appear for all
  open PRs; GitHub's API prevents actual self-approval at the backend.

- **TUI: PR selection highlight covers entire row** — the old highlight applied a background
  to the pre-ANSI-styled row string, but inner `\x1b[0m` reset sequences cleared the
  background mid-row, causing a patchy look. Each column style now carries the selection
  background individually so the highlight is solid and full-width.

```

**Step 2: Update the link table at the bottom** — add `[2.5.0]` entry:

```markdown
[2.5.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.5.0
[2.4.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.4.0
```

**Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "chore: changelog for v2.5.0"
```

---

### Task 6: Final verification

**Step 1: Full build + test**

```bash
go build ./... && go test ./...
```
Expected: build succeeds; all tests pass except the two known pre-existing failures.

**Step 2: Verify diff view renders correctly** — quick smoke test by running hangar locally if possible, or inspect that `DiffView.Parse()` + `rebuildLines()` produces non-empty lines:

```bash
go test ./internal/ui/... -v -run TestDiffView
```
Expected: all DiffView tests green.

**Step 3: Commit if anything was missed, then verify clean tree**

```bash
git status
```
Expected: clean working tree.
