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

	// lines is the flat list of rendered lines, rebuilt on Parse/SetSize.
	lines []renderedLine
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
// file whose header is at or above the current scroll position.
func (dv *DiffView) FileUnderCursor() (string, int) {
	if len(dv.lines) == 0 {
		return "", 0
	}
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

func (dv *DiffView) View() string {
	if !dv.visible {
		return ""
	}

	headerStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	footerStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	separatorStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	sep := separatorStyle.Render(strings.Repeat("─", max(dv.width-2, 0)))

	var b strings.Builder
	b.WriteString(headerStyle.Render("  Diff: "+dv.Summary()) + "\n")
	b.WriteString(sep + "\n")

	contentHeight := dv.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	if len(dv.lines) == 0 {
		b.WriteString(dimStyle.Render("  nothing to diff") + "\n")
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
	b.WriteString(footerStyle.Render("  j/k scroll · space/b page · ctrl+d/u half-page · g/G top/bottom · e editor · q close"))

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
	case " ", "f", "ctrl+f", "pgdown":
		// Full page down — less/more convention
		dv.ScrollDown(fullPage)
		return true, nil
	case "b", "ctrl+b", "pgup":
		// Full page up — less/more convention
		dv.ScrollUp(fullPage)
		return true, nil
	case "d", "ctrl+d":
		// Half page down
		dv.ScrollDown(halfPage)
		return true, nil
	case "u", "ctrl+u":
		// Half page up
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

// rebuildLines rebuilds the flat rendered-line cache from dv.files.
func (dv *DiffView) rebuildLines() {
	dv.lines = nil
	for _, f := range dv.files {
		path := strings.TrimPrefix(f.NewName, "b/")
		if path == "" || path == "/dev/null" {
			path = strings.TrimPrefix(f.OrigName, "a/")
		}

		firstLine := 0
		if len(f.Hunks) > 0 {
			firstLine = int(f.Hunks[0].NewStartLine)
		}

		dv.lines = append(dv.lines, renderedLine{
			text:     renderFileHeader(path),
			filePath: path,
			line:     firstLine,
		})

		for _, h := range f.Hunks {
			dv.lines = append(dv.lines, renderedLine{text: renderHunkHeader(h)})
			for _, bodyLine := range strings.Split(string(h.Body), "\n") {
				dv.lines = append(dv.lines, renderedLine{text: renderDiffLine(bodyLine)})
			}
		}
		dv.lines = append(dv.lines, renderedLine{text: ""})
	}
}

func renderFileHeader(path string) string {
	return lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("  " + path)
}

func renderHunkHeader(h *diff.Hunk) string {
	hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OrigStartLine, h.OrigLines, h.NewStartLine, h.NewLines)
	if h.Section != "" {
		hdr += " " + h.Section
	}
	return lipgloss.NewStyle().Foreground(ColorComment).Render("  " + hdr)
}

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
