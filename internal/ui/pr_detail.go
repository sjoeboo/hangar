package ui

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	prpkg "github.com/sjoeboo/hangar/internal/pr"
)

// PRDetailOverlay is a full-screen overlay showing PR detail with three tabs:
// Overview (metadata + body + file list), Diff (unified diff), and
// Conversation (comments and reviews in chronological order).
// It follows the same Show/Hide/IsVisible/SetSize/HandleKey contract as
// other overlays (DiffView, HelpOverlay, etc.) in home.go.
// diffFileEntry holds per-file state for the collapsible Diff tab.
type diffFileEntry struct {
	path      string
	status    string
	additions int
	deletions int
	hunk      string // raw diff section for this file
	expanded  bool
}

type PRDetailOverlay struct {
	visible bool
	width   int
	height  int

	pr      *prpkg.PR
	detail  *prpkg.PRDetail
	loading bool
	err     error

	tab         int // 0=Overview, 1=Description, 2=Diff, 3=Conversation
	scrollOffset int

	// lines is the flat rendered-line cache for the current tab, rebuilt by rebuildLines.
	lines []string

	// Diff tab state — reset when detail changes.
	diffFiles          []diffFileEntry
	diffFileCursor     int
	diffHeaderLineIdxs []int // which indices in lines[] are file headers
}

func NewPRDetailOverlay() *PRDetailOverlay {
	return &PRDetailOverlay{}
}

// Show opens the overlay for the given PR in loading state.
func (o *PRDetailOverlay) Show(pr *prpkg.PR) {
	o.pr = pr
	o.detail = nil
	o.loading = true
	o.err = nil
	o.tab = 0
	o.scrollOffset = 0
	o.lines = nil
	o.diffFiles = nil
	o.diffFileCursor = 0
	o.visible = true
}

func (o *PRDetailOverlay) Hide() {
	o.visible = false
	o.pr = nil
	o.detail = nil
}

func (o *PRDetailOverlay) IsVisible() bool {
	return o.visible
}

func (o *PRDetailOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
	o.rebuildLines()
}

// SetDetail is called when the async fetch completes (may be called with err != nil).
func (o *PRDetailOverlay) SetDetail(detail *prpkg.PRDetail, err error) {
	o.loading = false
	o.detail = detail
	o.err = err
	o.scrollOffset = 0
	o.diffFiles = nil // reset so buildDiffLines re-initialises with fresh file list
	o.diffFileCursor = 0
	o.rebuildLines()
}

// View renders the overlay as a full-screen string.
func (o *PRDetailOverlay) View() string {
	if !o.visible {
		return ""
	}

	var b strings.Builder

	// ── Header ──────────────────────────────────────────────────────────
	headerStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	title := "PR"
	if o.pr != nil {
		title = fmt.Sprintf("PR #%s: %s", prpkg.NumberStr(o.pr.Number), o.pr.Title)
	}
	runes := []rune(title)
	maxTitleW := o.width - 2
	if maxTitleW > 0 && len(runes) > maxTitleW {
		title = string(runes[:maxTitleW-1]) + "…"
	}
	b.WriteString(headerStyle.Render("  "+title) + "\n")

	// ── Tab bar ──────────────────────────────────────────────────────────
	tabNames := []string{"Overview", "Description", "Diff", "Conversation"}
	activeTabStyle := lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent).Bold(true).Padding(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(ColorComment).Padding(0, 1)
	var tabParts []string
	for i, name := range tabNames {
		if i == o.tab {
			tabParts = append(tabParts, activeTabStyle.Render(name))
		} else {
			tabParts = append(tabParts, inactiveTabStyle.Render(name))
		}
	}
	b.WriteString(strings.Join(tabParts, "") + "\n")

	separatorStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	sep := separatorStyle.Render(strings.Repeat("─", max(o.width-2, 0)))
	b.WriteString(sep + "\n")

	// ── Content ──────────────────────────────────────────────────────────
	// header(1) + tabs(1) + sep(1) + sep(1) + footer(2) = 6
	contentHeight := o.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}

	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	if o.loading {
		b.WriteString(dimStyle.Render("  Loading…") + "\n")
	} else if o.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(ColorRed)
		b.WriteString(errStyle.Render("  Error: "+o.err.Error()) + "\n")
	} else if len(o.lines) == 0 {
		b.WriteString(dimStyle.Render("  (empty)") + "\n")
	} else {
		start := o.scrollOffset
		if start < 0 {
			start = 0
		}
		if start >= len(o.lines) {
			start = len(o.lines) - 1
		}
		end := start + contentHeight
		if end > len(o.lines) {
			end = len(o.lines)
		}
		for _, line := range o.lines[start:end] {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString(sep + "\n")
	footerStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	hint := "  Tab/Shift+Tab switch tab · j/k scroll · g/G top/bottom · o browser · c comment · q close"
	if o.tab == 2 && len(o.diffFiles) > 0 {
		hint = "  j/k navigate files · enter toggle · d/u half-page scroll · g/G top/bottom · o browser · q close"
	}
	b.WriteString(footerStyle.Render(hint))

	// Pad to full height with background — do NOT use Width()+Height() on the
	// outer container because lipgloss will re-wrap and center ANSI-coded lines.
	content := b.String()
	bgStyle := lipgloss.NewStyle().Background(ColorBg)
	lines := strings.Split(content, "\n")
	// pad to o.height lines
	for len(lines) < o.height {
		lines = append(lines, "")
	}
	// pad each line to o.width
	for i, l := range lines {
		visible := lipgloss.Width(l)
		if visible < o.width {
			lines[i] = l + strings.Repeat(" ", o.width-visible)
		}
	}
	return bgStyle.Render(strings.Join(lines[:o.height], "\n"))
}

// HandleKey processes a key press. Returns (handled bool, cmd tea.Cmd).
func (o *PRDetailOverlay) HandleKey(key string) (bool, tea.Cmd) {
	if !o.visible {
		return false, nil
	}

	contentHeight := o.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}
	halfPage := contentHeight / 2
	if halfPage < 1 {
		halfPage = 1
	}

	// Diff tab: j/k navigate between file headers; enter/space toggle expand.
	if o.tab == 2 && len(o.diffFiles) > 0 {
		switch key {
		case "j", "down":
			if o.diffFileCursor < len(o.diffFiles)-1 {
				o.diffFileCursor++
				o.rebuildLines()
				o.scrollToDiffCursor()
			}
			return true, nil
		case "k", "up":
			if o.diffFileCursor > 0 {
				o.diffFileCursor--
				o.rebuildLines()
				o.scrollToDiffCursor()
			}
			return true, nil
		case "enter", " ":
			o.diffFiles[o.diffFileCursor].expanded = !o.diffFiles[o.diffFileCursor].expanded
			o.rebuildLines()
			o.scrollToDiffCursor()
			return true, nil
		}
	}

	switch key {
	case "q", "esc":
		o.Hide()
		return true, nil
	case "o":
		if o.pr != nil && o.pr.URL != "" {
			exec.Command("open", o.pr.URL).Start() //nolint:errcheck
		}
		return true, nil
	case "c":
		if o.pr != nil {
			pr := o.pr
			return true, func() tea.Msg { return prDetailCommentRequestMsg{pr: pr} }
		}
		return true, nil
	case "tab":
		o.tab = (o.tab + 1) % 4
		o.scrollOffset = 0
		o.rebuildLines()
		return true, nil
	case "shift+tab":
		o.tab = (o.tab + 3) % 4
		o.scrollOffset = 0
		o.rebuildLines()
		return true, nil
	case "j", "down":
		o.scrollDown(1)
		return true, nil
	case "k", "up":
		o.scrollUp(1)
		return true, nil
	case " ", "f", "ctrl+f", "pgdown":
		o.scrollDown(contentHeight)
		return true, nil
	case "b", "ctrl+b", "pgup":
		o.scrollUp(contentHeight)
		return true, nil
	case "d", "ctrl+d":
		o.scrollDown(halfPage)
		return true, nil
	case "u", "ctrl+u":
		o.scrollUp(halfPage)
		return true, nil
	case "g":
		o.scrollOffset = 0
		return true, nil
	case "G":
		o.scrollToBottom()
		return true, nil
	}
	return false, nil
}

func (o *PRDetailOverlay) scrollDown(n int) {
	contentHeight := o.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}
	limit := len(o.lines) - contentHeight
	if limit < 0 {
		limit = 0
	}
	o.scrollOffset += n
	if o.scrollOffset > limit {
		o.scrollOffset = limit
	}
}

func (o *PRDetailOverlay) scrollUp(n int) {
	o.scrollOffset -= n
	if o.scrollOffset < 0 {
		o.scrollOffset = 0
	}
}

func (o *PRDetailOverlay) scrollToBottom() {
	contentHeight := o.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}
	limit := len(o.lines) - contentHeight
	if limit < 0 {
		limit = 0
	}
	o.scrollOffset = limit
}

// scrollToDiffCursor scrolls so the currently focused diff file header is visible.
func (o *PRDetailOverlay) scrollToDiffCursor() {
	if o.diffFileCursor >= len(o.diffHeaderLineIdxs) {
		return
	}
	contentHeight := o.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}
	headerLine := o.diffHeaderLineIdxs[o.diffFileCursor]
	if headerLine < o.scrollOffset {
		o.scrollOffset = headerLine
	} else if headerLine >= o.scrollOffset+contentHeight {
		o.scrollOffset = headerLine - contentHeight + 1
	}
}

// rebuildLines regenerates the flat line cache for the current tab.
func (o *PRDetailOverlay) rebuildLines() {
	o.lines = nil
	if o.detail == nil {
		return
	}
	switch o.tab {
	case 0:
		o.lines = o.buildOverviewLines()
	case 1:
		o.lines = o.buildDescriptionLines()
	case 2:
		o.lines = o.buildDiffLines()
	case 3:
		o.lines = o.buildConversationLines()
	}
}

func (o *PRDetailOverlay) buildOverviewLines() []string {
	d := o.detail
	labelStyle := lipgloss.NewStyle().Foreground(ColorComment)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	field := func(label, value string) string {
		return labelStyle.Render(fmt.Sprintf("  %-18s", label)) + valueStyle.Render(value)
	}

	stateColor := ColorGreen
	switch d.State {
	case "MERGED":
		stateColor = ColorPurple
	case "CLOSED":
		stateColor = ColorRed
	case "DRAFT":
		stateColor = ColorComment
	}

	var lines []string
	lines = append(lines, field("State:", lipgloss.NewStyle().Foreground(stateColor).Render(d.State)))
	lines = append(lines, field("Author:", d.Author))
	lines = append(lines, field("Branch:", d.HeadBranch+" → "+d.BaseBranch))

	if d.ReviewDecision != "" {
		rdColor := ColorComment
		switch d.ReviewDecision {
		case "APPROVED":
			rdColor = ColorGreen
		case "CHANGES_REQUESTED":
			rdColor = ColorRed
		}
		lines = append(lines, field("Review:", lipgloss.NewStyle().Foreground(rdColor).Render(d.ReviewDecision)))
	}

	if d.Mergeability != "" {
		mColor := ColorComment
		switch d.Mergeability {
		case "MERGEABLE":
			mColor = ColorGreen
		case "CONFLICTING":
			mColor = ColorRed
		}
		lines = append(lines, field("Mergeability:", lipgloss.NewStyle().Foreground(mColor).Render(d.Mergeability)))
	}

	if d.HasChecks {
		checkStr := fmt.Sprintf("✓%d", d.ChecksPassed)
		if d.ChecksFailed > 0 {
			checkStr += fmt.Sprintf("  ✗%d", d.ChecksFailed)
		}
		if d.ChecksPending > 0 {
			checkStr += fmt.Sprintf("  ●%d", d.ChecksPending)
		}
		lines = append(lines, field("Checks:", checkStr))
	}

	lines = append(lines, "")

	// Files changed
	if len(d.Files) > 0 {
		hdrStyle := lipgloss.NewStyle().Foreground(ColorComment).Bold(true)
		lines = append(lines, hdrStyle.Render(fmt.Sprintf("  Files Changed (%d)", len(d.Files))))
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", max(o.width-6, 4))))
		for _, f := range d.Files {
			statusColor := ColorText
			switch f.Status {
			case "added":
				statusColor = ColorGreen
			case "deleted":
				statusColor = ColorRed
			case "modified":
				statusColor = ColorYellow
			}
			diffStr := fmt.Sprintf("+%d -%d", f.Additions, f.Deletions)
			lines = append(lines,
				"  "+lipgloss.NewStyle().Foreground(statusColor).Render(f.Path)+
					"  "+lipgloss.NewStyle().Foreground(ColorComment).Render(diffStr))
		}
	}

	return lines
}

func (o *PRDetailOverlay) buildDescriptionLines() []string {
	d := o.detail
	if d.Body == "" {
		return []string{lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true).Render("  No description")}
	}

	// Render markdown via glamour — handles GFM, left-aligns naturally.
	wrapWidth := o.width - 4
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	if r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(wrapWidth),
	); err == nil {
		if rendered, rerr := r.Render(d.Body); rerr == nil {
			var lines []string
			for _, l := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				lines = append(lines, l)
			}
			return lines
		}
	}

	// Fallback: plain text if glamour fails.
	bodyStyle := lipgloss.NewStyle().Foreground(ColorText)
	var lines []string
	for _, line := range strings.Split(d.Body, "\n") {
		for _, wl := range prDetailWrapLine(line, o.width-4) {
			lines = append(lines, bodyStyle.Render("  "+wl))
		}
	}
	return lines
}

func (o *PRDetailOverlay) buildDiffLines() []string {
	d := o.detail
	if d.DiffContent == "" {
		return []string{lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true).Render("  No diff available")}
	}

	hunkMap := parseDiffHunks(d.DiffContent)

	// Initialise per-file state on first build (or after a detail reset).
	if o.diffFiles == nil {
		o.diffFiles = make([]diffFileEntry, 0, len(d.Files))
		autoExpand := len(d.Files) == 1
		for _, f := range d.Files {
			o.diffFiles = append(o.diffFiles, diffFileEntry{
				path:      f.Path,
				status:    f.Status,
				additions: f.Additions,
				deletions: f.Deletions,
				hunk:      hunkMap[f.Path],
				expanded:  autoExpand,
			})
		}
		o.diffFileCursor = 0
	}

	// Styles
	summaryStyle := lipgloss.NewStyle().Foreground(ColorComment)
	indicatorStyle := lipgloss.NewStyle().Foreground(ColorComment)
	statsStyle := lipgloss.NewStyle().Foreground(ColorComment)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	focusBg := lipgloss.Color("#1e2d3a")

	statusColor := func(status string) lipgloss.Color {
		switch status {
		case "added":
			return ColorGreen
		case "deleted", "removed":
			return ColorRed
		case "modified":
			return ColorYellow
		default:
			return ColorAccent
		}
	}

	// Summary bar
	totalAdd, totalDel := 0, 0
	for _, f := range d.Files {
		totalAdd += f.Additions
		totalDel += f.Deletions
	}
	var lines []string
	lines = append(lines, summaryStyle.Render(fmt.Sprintf("  %d file%s changed  +%d -%d",
		len(d.Files), map[bool]string{true: "s", false: ""}[len(d.Files) != 1], totalAdd, totalDel)))
	lines = append(lines, "")

	o.diffHeaderLineIdxs = nil

	for i, entry := range o.diffFiles {
		headerIdx := len(lines)
		o.diffHeaderLineIdxs = append(o.diffHeaderLineIdxs, headerIdx)

		indicator := "▶"
		if entry.expanded {
			indicator = "▼"
		}
		focused := i == o.diffFileCursor

		pathStyle := lipgloss.NewStyle().Foreground(statusColor(entry.status))
		indStyle := indicatorStyle
		stStyle := statsStyle
		if focused {
			pathStyle = pathStyle.Background(focusBg)
			indStyle = indStyle.Background(focusBg)
			stStyle = stStyle.Background(focusBg)
		}

		// Truncate path if needed
		maxPathW := o.width - 20
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
			// Pad header to full width so focus background extends across the row.
			visible := lipgloss.Width(header)
			if visible < o.width-2 {
				header += lipgloss.NewStyle().Background(focusBg).Render(strings.Repeat(" ", o.width-2-visible))
			}
		}
		lines = append(lines, header)

		if entry.expanded {
			if entry.hunk != "" {
				hunkLines := strings.Split(entry.hunk, "\n")
				// Skip the "diff --git", "index", "---", "+++" header lines (first 4).
				start := 4
				if start > len(hunkLines) {
					start = len(hunkLines)
				}
				for _, l := range hunkLines[start:] {
					lines = append(lines, "  "+renderDiffLine(l))
				}
			} else {
				lines = append(lines, dimStyle.Render("    (no diff available for this file)"))
			}
			lines = append(lines, "") // blank separator after expanded content
		}
	}

	return lines
}

// parseDiffHunks splits a unified diff into per-file sections keyed by file path.
func parseDiffHunks(diff string) map[string]string {
	result := make(map[string]string)
	rawLines := strings.Split(diff, "\n")
	fileStart := -1
	currentPath := ""
	for i, line := range rawLines {
		if strings.HasPrefix(line, "diff --git ") {
			if fileStart >= 0 && currentPath != "" {
				result[currentPath] = strings.Join(rawLines[fileStart:i], "\n")
			}
			// "diff --git a/<path> b/<path>" — take the b/ path.
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentPath = strings.TrimPrefix(parts[3], "b/")
			}
			fileStart = i
		}
	}
	if fileStart >= 0 && currentPath != "" {
		result[currentPath] = strings.Join(rawLines[fileStart:], "\n")
	}
	return result
}

func (o *PRDetailOverlay) buildConversationLines() []string {
	d := o.detail

	type entry struct {
		t     int64 // unix nano for sorting
		lines []string
	}

	var entries []entry

	authorStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(ColorComment)

	// Comments
	for _, c := range d.Comments {
		header := "  " + authorStyle.Render(c.Author) + "  " + timeStyle.Render(c.CreatedAt.Format("Jan 02 15:04"))
		var ls []string
		ls = append(ls, header)
		ls = append(ls, o.renderMarkdownBlock(c.Body, o.width-6)...)
		ls = append(ls, "")
		entries = append(entries, entry{t: c.CreatedAt.UnixNano(), lines: ls})
	}

	// Reviews
	for _, r := range d.Reviews {
		stateColor := ColorComment
		switch r.State {
		case "APPROVED":
			stateColor = ColorGreen
		case "CHANGES_REQUESTED":
			stateColor = ColorRed
		}
		stateStyle := lipgloss.NewStyle().Foreground(stateColor)
		header := "  " + authorStyle.Render(r.Author) + " " +
			stateStyle.Render("["+r.State+"]") + "  " +
			timeStyle.Render(r.CreatedAt.Format("Jan 02 15:04"))
		var ls []string
		ls = append(ls, header)
		if r.Body != "" {
			ls = append(ls, o.renderMarkdownBlock(r.Body, o.width-6)...)
		}
		for _, c := range r.Comments {
			pathStyle := lipgloss.NewStyle().Foreground(ColorYellow)
			ls = append(ls, "    "+pathStyle.Render(fmt.Sprintf("%s:%d", c.Path, c.Line)))
			ls = append(ls, o.renderMarkdownBlock(c.Body, o.width-8)...)
		}
		ls = append(ls, "")
		entries = append(entries, entry{t: r.CreatedAt.UnixNano(), lines: ls})
	}

	if len(entries) == 0 {
		return []string{lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true).Render("  No conversation yet")}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].t < entries[j].t })

	var lines []string
	for _, e := range entries {
		lines = append(lines, e.lines...)
	}
	return lines
}

// renderMarkdownBlock renders text as GFM markdown using glamour, returning
// one element per output line. Falls back to plain-text wrapping on error.
func (o *PRDetailOverlay) renderMarkdownBlock(text string, wrapWidth int) []string {
	if text == "" {
		return nil
	}
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	if r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(wrapWidth),
	); err == nil {
		if rendered, rerr := r.Render(text); rerr == nil {
			var lines []string
			for _, l := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				lines = append(lines, l)
			}
			return lines
		}
	}
	// Fallback: plain text
	textStyle := lipgloss.NewStyle().Foreground(ColorText)
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		for _, wl := range prDetailWrapLine(l, wrapWidth) {
			lines = append(lines, textStyle.Render("  "+wl))
		}
	}
	return lines
}

// prDetailWrapLine wraps s to fit within maxWidth runes per line.
func prDetailWrapLine(s string, maxWidth int) []string {
	if maxWidth <= 0 || len(s) == 0 {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return []string{s}
	}
	var result []string
	for len(runes) > maxWidth {
		result = append(result, string(runes[:maxWidth]))
		runes = runes[maxWidth:]
	}
	if len(runes) > 0 {
		result = append(result, string(runes))
	}
	return result
}
