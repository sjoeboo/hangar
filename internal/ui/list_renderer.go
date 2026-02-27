package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ghe.spotify.net/mnicholson/hangar/internal/session"
)

// Tree drawing characters for visual hierarchy
const (
	treeBranch = "├─" // Mid-level item (has siblings below)
	treeLast   = "└─" // Last item in group (no siblings below)
	treeEmpty  = "  " // Empty space (for alignment)
	// Sub-session connectors (nested under parent)
	subBranch = "├─" // Sub-session with siblings below
	subLast   = "└─" // Last sub-session
)

// renderSessionList renders the left panel session list.
func (h *Home) renderSessionList(width, height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 {
		// Responsive empty state - adapts to available space
		// Account for border (2 chars each side) when calculating content area
		contentWidth := width - 4
		contentHeight := height - 2
		if contentWidth < 10 {
			contentWidth = 10
		}
		if contentHeight < 5 {
			contentHeight = 5
		}

		emptyContent := renderEmptyStateResponsive(EmptyStateConfig{
			Icon:     "⬡",
			Title:    "No Sessions Yet",
			Subtitle: "Get started by creating your first session",
			Hints: []string{
				"Press n to create a new session",
				"Press i to import existing tmux sessions",
				"Press g to create a group",
			},
		}, contentWidth, contentHeight)

		return styleListEmptyBorder.Render(emptyContent)
	}

	// Render items starting from viewOffset
	visibleCount := 0
	maxVisible := height - 1 // Leave room for scrolling indicator
	if maxVisible < 1 {
		maxVisible = 1
	}

	// Show "more above" indicator if scrolled down
	if h.viewOffset > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d above", h.viewOffset)))
		b.WriteString("\n")
		maxVisible-- // Account for the indicator line
	}

	for i := h.viewOffset; i < len(h.flatItems) && visibleCount < maxVisible; i++ {
		item := h.flatItems[i]
		h.renderItem(&b, item, i == h.cursor, i)
		visibleCount++
	}

	// Show "more below" indicator if there are more items
	remaining := len(h.flatItems) - (h.viewOffset + visibleCount)
	if remaining > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d below", remaining)))
	}

	// Height padding is handled by ensureExactHeight() in View() for consistency
	return b.String()
}

// renderItem renders a single item (group or session) for the left panel
func (h *Home) renderItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	if item.Type == session.ItemTypeGroup {
		h.renderGroupItem(b, item, selected, itemIndex)
	} else {
		h.renderSessionItem(b, item, selected)
	}
}

// renderGroupItem renders a group header
// PERFORMANCE: Uses cached styles from styles.go to avoid allocations
func (h *Home) renderGroupItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	group := item.Group

	// Calculate indentation based on nesting level (no tree lines, just spaces)
	// Uses spacingNormal (2 chars) per level for consistent hierarchy visualization
	indent := strings.Repeat(strings.Repeat(" ", spacingNormal), max(0, item.Level))

	// Expand/collapse indicator with filled triangles (using cached styles)
	var expandIcon string
	if selected {
		if group.Expanded {
			expandIcon = GroupExpandSelStyle.Render("▾")
		} else {
			expandIcon = GroupExpandSelStyle.Render("▸")
		}
	} else {
		if group.Expanded {
			expandIcon = GroupExpandStyle.Render("▾") // Filled triangle for expanded
		} else {
			expandIcon = GroupExpandStyle.Render("▸") // Filled triangle for collapsed
		}
	}

	// Hotkey indicator (subtle, only for root groups, hidden when selected)
	// Uses pre-computed RootGroupNum from rebuildFlatItems() - O(1) lookup instead of O(n) loop
	hotkeyStr := ""
	if item.Level == 0 && !selected {
		if item.RootGroupNum >= 1 && item.RootGroupNum <= 9 {
			hotkeyStr = GroupHotkeyStyle.Render(fmt.Sprintf("%d·", item.RootGroupNum))
		}
	}

	// Select appropriate cached styles based on selection state
	nameStyle := GroupNameStyle
	countStyle := GroupCountStyle
	if selected {
		nameStyle = GroupNameSelStyle
		countStyle = GroupCountSelStyle
	}

	// Use recursive count to include sessions in subgroups (Issue #48)
	sessionCount := h.groupTree.SessionCountForGroup(group.Path)
	countStr := countStyle.Render(fmt.Sprintf(" (%d)", sessionCount))

	// Status indicators (compact, on same line) using cached styles
	// Also count recursively for subgroups
	running := 0
	waiting := 0
	for path, g := range h.groupTree.Groups {
		if path == group.Path || strings.HasPrefix(path, group.Path+"/") {
			for _, sess := range g.Sessions {
				switch sess.Status {
				case session.StatusRunning:
					running++
				case session.StatusWaiting:
					waiting++
				}
			}
		}
	}

	statusStr := ""
	if running > 0 {
		statusStr += " " + GroupStatusRunning.Render(fmt.Sprintf("● %d", running))
	}
	if waiting > 0 {
		statusStr += " " + GroupStatusWaiting.Render(fmt.Sprintf("◐ %d", waiting))
	}

	// Build the row: [indent][hotkey][expand] [name](count) [status]
	row := fmt.Sprintf("%s%s%s %s%s%s", indent, hotkeyStr, expandIcon, nameStyle.Render(group.Name), countStr, statusStr)
	b.WriteString(row)
	b.WriteString("\n")
}

// renderSessionItem renders a single session item for the left panel
// PERFORMANCE: Uses cached styles from styles.go to avoid allocations
func (h *Home) renderSessionItem(b *strings.Builder, item session.Item, selected bool) {
	inst := item.Session

	// Snapshot status and tool under read lock to avoid races with background worker
	instStatus := inst.GetStatusThreadSafe()
	instTool := inst.GetToolThreadSafe()

	// Tree style for connectors - Use ColorText for clear visibility of box-drawing characters
	treeStyle := TreeConnectorStyle

	// Calculate base indentation for parent levels
	// Level 1 means direct child of root group, Level 2 means child of nested group, etc.
	baseIndent := ""
	if item.Level > 1 {
		// For deeply nested items, add spacing for parent levels
		// Sub-sessions get extra indentation (they're at Level = groupLevel + 2)
		if item.IsSubSession {
			// Sub-session: indent for group level, then continuation line for parent
			// Add leading space so │ aligns with ├ in regular items (both at position 1)
			groupIndent := strings.Repeat(treeEmpty, item.Level-2)
			if item.ParentIsLastInGroup {
				baseIndent = groupIndent + "  " // 2 spaces - parent is last, no continuation needed
			} else {
				// Style the │ character - leading space aligns │ with ├ above
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		} else {
			baseIndent = strings.Repeat(treeEmpty, item.Level-1)
		}
	}

	// Tree connector: └─ for last item, ├─ for others
	treeConnector := treeBranch
	if item.IsSubSession {
		// Sub-session uses its own last-in-group logic
		if item.IsLastSubSession {
			treeConnector = subLast
		} else {
			treeConnector = subBranch
		}
	} else if item.IsLastInGroup {
		treeConnector = treeLast
	}

	// Status indicator with consistent sizing
	var statusIcon string
	var statusStyle lipgloss.Style
	switch instStatus {
	case session.StatusRunning:
		statusIcon = "●"
		statusStyle = SessionStatusRunning
	case session.StatusWaiting:
		statusIcon = "◐"
		statusStyle = SessionStatusWaiting
	case session.StatusIdle:
		statusIcon = "○"
		statusStyle = SessionStatusIdle
	case session.StatusError:
		statusIcon = "✕"
		statusStyle = SessionStatusError
	default:
		statusIcon = "○"
		statusStyle = SessionStatusIdle
	}

	status := statusStyle.Render(statusIcon)

	// Title styling - add bold/underline for accessibility (colorblind users)
	var titleStyle lipgloss.Style
	switch instStatus {
	case session.StatusRunning, session.StatusWaiting:
		// Bold for active states (distinguishable without color)
		titleStyle = SessionTitleActive
	case session.StatusError:
		// Underline for error (distinguishable without color)
		titleStyle = SessionTitleError
	default:
		titleStyle = SessionTitleDefault
	}

	// Tool badge with brand-specific color
	// Claude=orange, Gemini=purple, Codex=cyan, Aider=red
	toolStyle := GetToolStyle(instTool)

	// Selection indicator
	var selectionPrefix string
	if h.bulkSelectMode {
		// In bulk mode: show checkbox, apply cursor highlight independently
		if h.selectedSessionIDs[inst.ID] {
			selectionPrefix = SessionSelectionPrefix.Render("☑")
		} else {
			selectionPrefix = SessionCheckboxUnchecked.Render("□")
		}
		if selected {
			// Apply cursor highlight styling (background color) for focused row
			titleStyle = SessionTitleSelStyle
			toolStyle = SessionStatusSelStyle
			statusStyle = SessionStatusSelStyle
			status = statusStyle.Render(statusIcon)
			treeStyle = TreeConnectorSelStyle
			if item.IsSubSession && !item.ParentIsLastInGroup {
				groupIndent := strings.Repeat(treeEmpty, max(0, item.Level-2))
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		}
	} else {
		// Normal mode: arrow cursor indicator
		selectionPrefix = " "
		if selected {
			selectionPrefix = SessionSelectionPrefix.Render("▶")
			titleStyle = SessionTitleSelStyle
			toolStyle = SessionStatusSelStyle
			statusStyle = SessionStatusSelStyle
			status = statusStyle.Render(statusIcon)
			// Tree connector also gets selection styling
			treeStyle = TreeConnectorSelStyle
			// Rebuild baseIndent with selection styling for sub-sessions
			if item.IsSubSession && !item.ParentIsLastInGroup {
				groupIndent := strings.Repeat(treeEmpty, max(0, item.Level-2))
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		}
	}

	title := titleStyle.Render(inst.Title)
	tool := ""
	if instTool != "claude" {
		tool = toolStyle.Render(" " + instTool)
	}

	// YOLO badge for Gemini sessions with YOLO mode enabled
	yoloBadge := ""
	if instTool == "gemini" && inst.GeminiYoloMode != nil && *inst.GeminiYoloMode {
		yoloStyle := styleYoloBadge
		if selected {
			yoloStyle = SessionStatusSelStyle
		}
		yoloBadge = yoloStyle.Render(" [YOLO]")
	}

	// PR badge for sessions with an open/merged/closed pull request
	prBadge := ""
	if inst.IsWorktree() {
		pr, hasPR := h.cache.GetPR(inst.ID)
		if hasPR && pr != nil {
			var badgeStyle lipgloss.Style
			validState := true
			switch pr.State {
			case "OPEN":
				badgeStyle = PRBadgeOpen
			case "MERGED":
				badgeStyle = PRBadgeMerged
			case "CLOSED":
				badgeStyle = PRBadgeClosed
			default:
				validState = false // DRAFT and unknown states: no badge
			}
			if validState {
				if selected {
					badgeStyle = SessionStatusSelStyle
				}
				prBadge = badgeStyle.Render(fmt.Sprintf(" [#%d]", pr.Number))
			}
		}
	}

	// Build row: [baseIndent][selection][tree][status] [title] [tool] [yolo] [pr]
	// Format: " ├─ ● session-name" or "▶└─ ● session-name"
	// Sub-sessions get extra indent: "   ├─◐ sub-session"
	row := fmt.Sprintf("%s%s%s %s %s%s%s%s", baseIndent, selectionPrefix, treeStyle.Render(treeConnector), status, title, tool, yoloBadge, prBadge)
	b.WriteString(row)
	b.WriteString("\n")
}
