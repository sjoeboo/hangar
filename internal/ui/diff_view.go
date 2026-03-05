package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sourcegraph/go-diff/diff"
)

// ── Pre-compiled styles (initialized by initStyles() after InitTheme sets colors) ──
var (
	// Pre-compiled styles for DiffView overlay and shared diff line rendering.
	diffViewHeaderStyle    lipgloss.Style
	diffViewSeparatorStyle lipgloss.Style
	diffViewFooterStyle    lipgloss.Style
	diffHunkHeaderStyle    lipgloss.Style
	diffLineAddStyle       lipgloss.Style
	diffLineDelStyle       lipgloss.Style
	diffLineContextStyle   lipgloss.Style
	diffLineNoNewlineStyle lipgloss.Style
)

// editorFinishedMsg is returned by the tea.ExecProcess callback when the
// editor process exits. The err field is non-nil if the editor exited with a
// non-zero status or could not be started.
type editorFinishedMsg struct{ err error }

// DiffView renders a parsed git diff as a scrollable full-screen overlay.
// It follows the same Show/Hide/IsVisible/SetSize contract as other overlays
// (HelpOverlay, GlobalSearch, etc.) in home.go.
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

// renderedLine holds one display line and the source file metadata it belongs to.
// filePath is non-empty only for file-header lines; used by FileUnderCursor.
type renderedLine struct {
	text     string
	filePath string // non-empty for file header lines only
	line     int    // first changed line in this file (0 if not a header)
}

func NewDiffView() *DiffView {
	return &DiffView{}
}

func (dv *DiffView) Show() {
	dv.visible = true
	dv.scrollOffset = 0
}

func (dv *DiffView) Hide() {
	dv.visible = false
}

func (dv *DiffView) IsVisible() bool {
	return dv.visible
}

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
		} else if strings.TrimPrefix(f.NewName, "b/") == "/dev/null" {
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
		})
	}

	dv.rebuildLines()
	return nil
}

func (dv *DiffView) FileCount() int {
	return len(dv.files)
}

// Summary returns a short human-readable summary, e.g. "3 files, +47 -12".
func (dv *DiffView) Summary() string {
	if len(dv.files) == 0 {
		return "no changes"
	}
	var additions, deletions int
	for _, f := range dv.files {
		for _, h := range f.Hunks {
			for _, bodyLine := range strings.Split(string(h.Body), "\n") {
				switch {
				case strings.HasPrefix(bodyLine, "+"):
					additions++
				case strings.HasPrefix(bodyLine, "-"):
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
// file currently under the cursor.
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

// ScrollDown scrolls down by n lines, clamped to end of content.
func (dv *DiffView) ScrollDown(n int) {
	visibleHeight := dv.height - 4
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	limit := len(dv.lines) - visibleHeight
	if limit < 0 {
		limit = 0
	}
	dv.scrollOffset += n
	if dv.scrollOffset > limit {
		dv.scrollOffset = limit
	}
}

// ScrollUp scrolls up by n lines, clamped to 0.
func (dv *DiffView) ScrollUp(n int) {
	dv.scrollOffset -= n
	if dv.scrollOffset < 0 {
		dv.scrollOffset = 0
	}
}

// ScrollToTop scrolls to the first line.
func (dv *DiffView) ScrollToTop() {
	dv.scrollOffset = 0
}

// ScrollToBottom scrolls to the last visible position.
func (dv *DiffView) ScrollToBottom() {
	visibleHeight := dv.height - 4
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	limit := len(dv.lines) - visibleHeight
	if limit < 0 {
		limit = 0
	}
	dv.scrollOffset = limit
}

// HandleKey processes a key press when the overlay is visible.
// Returns (handled bool, cmd tea.Cmd).
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

// openInEditor returns a tea.Cmd that opens the given file at the given line
// in $EDITOR (falls back to "vi").  $EDITOR may contain flags (e.g. "vim -u
// NONE"), so it is split on whitespace before constructing the exec.Command.
func openInEditor(path string, line int) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	var args []string
	if line > 0 {
		args = append(parts[1:], fmt.Sprintf("+%d", line), path)
	} else {
		args = append(parts[1:], path)
	}
	cmd := exec.Command(parts[0], args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// rebuildLines rebuilds the flat rendered-line cache from dv.diffFiles and dv.files.
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
		if i < len(dv.files) && len(dv.files[i].Hunks) > 0 {
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

// scrollToFileCursor adjusts scrollOffset so the cursor's file header is visible.
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

func renderHunkHeader(h *diff.Hunk) string {
	hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OrigStartLine, h.OrigLines, h.NewStartLine, h.NewLines)
	if h.Section != "" {
		hdr += " " + h.Section
	}
	return diffHunkHeaderStyle.Render("  " + hdr)
}

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
		// "\ No newline at end of file"
		return diffLineNoNewlineStyle.Render(line)
	default:
		return diffLineContextStyle.Render(line)
	}
}
