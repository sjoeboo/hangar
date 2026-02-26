package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sourcegraph/go-diff/diff"
)

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
	b.WriteString(footerStyle.Render("  j/k scroll · e open editor · q/esc close"))

	return lipgloss.NewStyle().
		Width(dv.width).
		Height(dv.height).
		Background(ColorBg).
		Padding(0, 1).
		Render(b.String())
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
