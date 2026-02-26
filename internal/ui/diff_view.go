package ui

import (
	"fmt"
	"strings"

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

// View is a stub — rendering is added in Task 4.
func (dv *DiffView) View() string {
	return ""
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

		// File header line — filePath is set so FileUnderCursor can find it
		dv.lines = append(dv.lines, renderedLine{
			text:     "  " + path,
			filePath: path,
			line:     firstLine,
		})

		for _, h := range f.Hunks {
			// Hunk header
			hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OrigStartLine, h.OrigLines, h.NewStartLine, h.NewLines)
			if h.Section != "" {
				hdr += " " + h.Section
			}
			dv.lines = append(dv.lines, renderedLine{text: "  " + hdr})

			// Hunk body lines
			for _, bodyLine := range strings.Split(string(h.Body), "\n") {
				dv.lines = append(dv.lines, renderedLine{text: bodyLine})
			}
		}

		// Blank separator between files
		dv.lines = append(dv.lines, renderedLine{text: ""})
	}
}
