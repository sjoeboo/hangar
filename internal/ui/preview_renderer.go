package ui

// preview_renderer.go — preview pane rendering for the Hangar TUI.
//
// Extracted from home.go as part of the incremental decomposition of that file.
// All functions here are *Home receiver methods or package-level helpers that are
// exclusively used by preview rendering.

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"ghe.spotify.net/mnicholson/hangar/internal/session"
	"ghe.spotify.net/mnicholson/hangar/internal/tmux"
)

// renderSectionDivider creates a modern section divider with optional centered label
// Format: ─────────── Label ─────────── (lines extend to fill width)
func renderSectionDivider(label string, width int) string {
	lineStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	if label == "" {
		return lineStyle.Render(strings.Repeat("─", max(0, width)))
	}

	// Label with subtle background for better visibility
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)

	// Calculate side widths
	labelWidth := len(label) + 2 // +2 for spacing on each side of label
	sideWidth := (width - labelWidth) / 2
	if sideWidth < 3 {
		sideWidth = 3
	}

	return lineStyle.Render(strings.Repeat("─", sideWidth)) +
		" " + labelStyle.Render(label) + " " +
		lineStyle.Render(strings.Repeat("─", sideWidth))
}

// renderToolStatusLine renders a Status + Session line for a tool section.
// sessionID is the detected session ID (empty = not connected).
// detectedAt is when detection ran (zero = still detecting, used only when threeState is true).
// threeState enables the "Detecting..." intermediate state (for tools like OpenCode/Codex).
func renderToolStatusLine(b *strings.Builder, sessionID string, detectedAt time.Time, threeState bool) {
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	if sessionID != "" {
		statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
		b.WriteString(labelStyle.Render("Status:  "))
		b.WriteString(statusStyle.Render("● Connected"))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Session: "))
		b.WriteString(valueStyle.Render(sessionID))
		b.WriteString("\n")
	} else if threeState && detectedAt.IsZero() {
		statusStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString(labelStyle.Render("Status:  "))
		b.WriteString(statusStyle.Render("◐ Detecting session..."))
		b.WriteString("\n")
	} else {
		statusStyle := lipgloss.NewStyle().Foreground(ColorText)
		b.WriteString(labelStyle.Render("Status:  "))
		if threeState {
			b.WriteString(statusStyle.Render("○ No session found"))
		} else {
			b.WriteString(statusStyle.Render("○ Not connected"))
		}
		b.WriteString("\n")
	}
}

// renderDetectedAtLine renders a "Detected: X ago" line.
func renderDetectedAtLine(b *strings.Builder, detectedAt time.Time) {
	if detectedAt.IsZero() {
		return
	}
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
	b.WriteString(labelStyle.Render("Detected:"))
	b.WriteString(dimStyle.Render(" " + formatRelativeTime(detectedAt)))
	b.WriteString("\n")
}

// renderForkHintLine renders the fork keyboard hint line.
func renderForkHintLine(b *strings.Builder) {
	hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
	keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	b.WriteString(hintStyle.Render("Fork:    "))
	b.WriteString(keyStyle.Render("f"))
	b.WriteString(hintStyle.Render(" quick fork, "))
	b.WriteString(keyStyle.Render("F"))
	b.WriteString(hintStyle.Render(" fork with options"))
	b.WriteString("\n")
}

// renderSimpleMCPLine renders MCPs without sync status (for Gemini and other tools).
// Width-aware truncation shows "(+N more)" when MCPs don't fit.
func renderSimpleMCPLine(b *strings.Builder, mcpInfo *session.MCPInfo, width int) {
	if mcpInfo == nil || !mcpInfo.HasAny() {
		return
	}

	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	var mcpParts []string
	for _, name := range mcpInfo.Global {
		mcpParts = append(mcpParts, valueStyle.Render(name+" (g)"))
	}
	for _, name := range mcpInfo.Project {
		mcpParts = append(mcpParts, valueStyle.Render(name+" (p)"))
	}
	for _, mcp := range mcpInfo.LocalMCPs {
		mcpParts = append(mcpParts, valueStyle.Render(mcp.Name+" (l)"))
	}

	if len(mcpParts) == 0 {
		return
	}

	b.WriteString(labelStyle.Render("MCPs:    "))

	mcpMaxWidth := width - 4 - 9
	if mcpMaxWidth < 20 {
		mcpMaxWidth = 20
	}

	var mcpResult strings.Builder
	mcpCount := 0
	currentWidth := 0

	for i, part := range mcpParts {
		plainPart := tmux.StripANSI(part)
		partWidth := runewidth.StringWidth(plainPart)

		addedWidth := partWidth
		if mcpCount > 0 {
			addedWidth += 2
		}

		remaining := len(mcpParts) - i
		isLast := remaining == 1

		var wouldExceed bool
		if isLast {
			wouldExceed = currentWidth+addedWidth > mcpMaxWidth
		} else {
			moreIndicator := fmt.Sprintf(" (+%d more)", remaining)
			moreWidth := runewidth.StringWidth(moreIndicator)
			wouldExceed = currentWidth+addedWidth+moreWidth > mcpMaxWidth
		}

		if wouldExceed {
			moreStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
			if mcpCount > 0 {
				mcpResult.WriteString(moreStyle.Render(fmt.Sprintf(" (+%d more)", remaining)))
			} else {
				mcpResult.WriteString(moreStyle.Render(fmt.Sprintf("(%d MCPs)", len(mcpParts))))
			}
			break
		}

		if mcpCount > 0 {
			mcpResult.WriteString(", ")
		}
		mcpResult.WriteString(part)
		currentWidth += addedWidth
		mcpCount++
	}

	b.WriteString(mcpResult.String())
	b.WriteString("\n")
}

// renderLaunchingState renders the animated launching/resuming indicator for sessions
func (h *Home) renderLaunchingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Check if this is a resume operation (vs new launch)
	_, isResuming := h.resumingSessions[inst.ID]

	// Braille spinner frames - creates smooth rotation effect
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Tool-specific messaging with emoji
	var toolName, toolDesc, emoji string
	if isResuming {
		emoji = "🔄"
	} else {
		emoji = "🚀"
	}

	switch inst.Tool {
	case "claude":
		toolName = "Claude Code"
		if isResuming {
			toolDesc = "Resuming Claude session..."
		} else {
			toolDesc = "Starting Claude session..."
		}
	case "gemini":
		toolName = "Gemini"
		if isResuming {
			toolDesc = "Resuming Gemini session..."
		} else {
			toolDesc = "Connecting to Gemini..."
		}
	case "aider":
		toolName = "Aider"
		if isResuming {
			toolDesc = "Resuming Aider session..."
		} else {
			toolDesc = "Starting Aider..."
		}
	case "codex":
		toolName = "Codex"
		if isResuming {
			toolDesc = "Resuming Codex session..."
		} else {
			toolDesc = "Starting Codex..."
		}
	case "opencode":
		toolName = "OpenCode"
		if isResuming {
			toolDesc = "Resuming OpenCode session..."
		} else {
			toolDesc = "Starting OpenCode..."
		}
	default:
		toolName = "Shell"
		if isResuming {
			toolDesc = "Resuming shell session..."
		} else {
			toolDesc = "Launching shell session..."
		}
	}

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Spinner with tool color
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// Title with emoji
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	var actionVerb string
	if isResuming {
		actionVerb = "Resuming"
	} else {
		actionVerb = "Launching"
	}
	b.WriteString(centerStyle.Render(titleStyle.Render(emoji + " " + actionVerb + " " + toolName)))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render(toolDesc)))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorAccent)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time (consistent with MCP and Fork animations)
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderMcpLoadingState renders the MCP loading animation in the preview pane
func (h *Home) renderMcpLoadingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Braille spinner frames - creates smooth rotation effect
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Spinner with cyan color (MCP-themed)
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// MCP loading title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	b.WriteString(centerStyle.Render(titleStyle.Render("🔌 Reloading MCPs")))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render("Restarting session with updated MCP configuration...")))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorCyan)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderForkingState renders the forking animation when session is being forked
func (h *Home) renderForkingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Braille spinner frames
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Spinner with purple color (fork-themed)
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// Forking title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	b.WriteString(centerStyle.Render(titleStyle.Render("🔀 Forking Session")))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render("Creating a new Claude session from this conversation...")))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorPurple)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time (consistent with other animations)
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderSessionInfoCard renders a simple session info card as fallback view
// Used when both show_output and show_analytics are disabled
func (h *Home) renderSessionInfoCard(inst *session.Instance, width, height int) string {
	if inst == nil {
		dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
		return dimStyle.Render("No session selected")
	}

	var b strings.Builder

	// Snapshot status/tool under read lock for thread safety
	cardStatus := inst.GetStatusThreadSafe()
	cardTool := inst.GetToolThreadSafe()

	// Header with tool icon
	icon := ToolIcon(cardTool)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Render(fmt.Sprintf("%s %s", icon, inst.Title))
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", max(0, min(width-4, 40))))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	// Path
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Path:"), valueStyle.Render(inst.ProjectPath)))

	// Status with color
	var statusColor lipgloss.Color
	switch cardStatus {
	case session.StatusRunning:
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusColor = ColorYellow
	case session.StatusError:
		statusColor = ColorRed
	default:
		statusColor = ColorTextDim
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), statusStyle.Render(string(cardStatus))))

	// Tool
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Tool:"), valueStyle.Render(cardTool)))

	// Session ID (if available) - Claude, Gemini, or OpenCode
	sessionID := inst.ClaudeSessionID
	if sessionID == "" {
		sessionID = inst.GeminiSessionID
	}
	if sessionID == "" {
		sessionID = inst.OpenCodeSessionID
	}
	if sessionID != "" {
		shortID := sessionID
		if len(shortID) > 12 {
			shortID = shortID[:12] + "..."
		}
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Session:"), valueStyle.Render(shortID)))
	}

	// Created date
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), valueStyle.Render(inst.CreatedAt.Format("Jan 2 15:04"))))

	return b.String()
}

// renderPreviewPane renders the right panel with live preview
func (h *Home) renderPreviewPane(width, height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 || h.cursor >= len(h.flatItems) {
		// Show different message when there are no sessions vs just no selection
		if len(h.flatItems) == 0 {
			return renderEmptyStateResponsive(EmptyStateConfig{
				Icon:     "✦",
				Title:    "Ready to Go",
				Subtitle: "Your workspace is set up",
				Hints: []string{
					"Press n to create your first session",
					"Press i to import tmux sessions",
				},
			}, width, height)
		}
		return renderEmptyStateResponsive(EmptyStateConfig{
			Icon:     "◇",
			Title:    "No Selection",
			Subtitle: "Select a session to preview",
			Hints:    nil,
		}, width, height)
	}

	item := h.flatItems[h.cursor]

	// If group is selected, show group info
	if item.Type == session.ItemTypeGroup {
		return h.renderGroupPreview(item.Group, width, height)
	}

	// Session preview
	selected := item.Session

	// Session info header box
	statusIcon := "○"
	statusColor := ColorTextDim
	switch selected.Status {
	case session.StatusRunning:
		statusIcon = "●"
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusIcon = "◐"
		statusColor = ColorYellow
	case session.StatusError:
		statusIcon = "✕"
		statusColor = ColorRed
	}

	// Header with session name and status
	statusBadge := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + string(selected.Status))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	b.WriteString(nameStyle.Render(selected.Title))
	b.WriteString("  ")
	b.WriteString(statusBadge)
	b.WriteString("\n")

	// Info lines: path and activity time
	infoStyle := lipgloss.NewStyle().Foreground(ColorText)
	pathStr := shortenPath(selected.ProjectPath, width-4)
	b.WriteString(infoStyle.Render("📁 " + pathStr))
	b.WriteString("\n")

	// Activity time - shows when session was last active
	activityTime := selected.GetLastActivityTime()
	activityStr := formatRelativeTime(activityTime)
	if selected.Status == session.StatusRunning {
		activityStr = "active now"
	}
	b.WriteString(infoStyle.Render("⏱ " + activityStr))
	b.WriteString("\n")

	toolBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorPurple).
		Padding(0, 1).
		Render(selected.Tool)
	groupBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorCyan).
		Padding(0, 1).
		Render(selected.GroupPath)
	b.WriteString(toolBadge)
	b.WriteString(" ")
	b.WriteString(groupBadge)
	b.WriteString("\n")

	// Worktree info section (for sessions running in git worktrees)
	if selected.IsWorktree() {
		wtHeader := renderSectionDivider("Worktree", width-4)
		b.WriteString(wtHeader)
		b.WriteString("\n")

		wtLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
		wtBranchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
		wtValueStyle := lipgloss.NewStyle().Foreground(ColorText)
		wtHintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
		wtKeyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

		// PR status (from gh CLI, lazy-cached with 60s TTL)
		if h.ghPath != "" {
			h.prCacheMu.Lock()
			pr, hasPR := h.prCache[selected.ID]
			_, hasTs := h.prCacheTs[selected.ID]
			h.prCacheMu.Unlock()

			if !hasTs || (hasTs && !hasPR) {
				b.WriteString(wtLabelStyle.Render("PR:      "))
				b.WriteString(lipgloss.NewStyle().Foreground(ColorComment).Render("checking..."))
				b.WriteString("\n")
			} else if pr != nil {
				stateStyle := lipgloss.NewStyle()
				stateLabel := strings.ToLower(pr.State)
				switch pr.State {
				case "OPEN":
					stateStyle = stateStyle.Foreground(ColorGreen)
				case "DRAFT":
					stateStyle = stateStyle.Foreground(ColorComment)
				case "MERGED":
					stateStyle = stateStyle.Foreground(ColorPurple)
				case "CLOSED":
					stateStyle = stateStyle.Foreground(ColorRed)
				}
				prNumStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Underline(true)
				titleMax := width - 4 - 9 - 6 - len(stateLabel) - 3
				title := pr.Title
				if titleMax > 10 && runewidth.StringWidth(title) > titleMax {
					title = runewidth.Truncate(title, titleMax, "...")
				}
				b.WriteString(wtLabelStyle.Render("PR:      "))
				b.WriteString(prNumStyle.Render(fmt.Sprintf("#%d", pr.Number)))
				b.WriteString(" ")
				b.WriteString(stateStyle.Render(stateLabel))
				b.WriteString(wtValueStyle.Render(" · " + title))
				b.WriteString("\n")
				if pr.URL != "" {
					urlStyle := lipgloss.NewStyle().Foreground(ColorComment)
					urlMax := width - 4 - 9
					displayURL := pr.URL
					if runewidth.StringWidth(displayURL) > urlMax && urlMax > 15 {
						displayURL = runewidth.Truncate(displayURL, urlMax, "…")
					}
					b.WriteString(wtLabelStyle.Render("         "))
					b.WriteString(urlStyle.Render(displayURL))
					b.WriteString("\n")
				}
				if pr.HasChecks {
					b.WriteString(wtLabelStyle.Render("Checks:  "))
					var parts []string
					if pr.ChecksFailed > 0 {
						parts = append(parts, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✗ %d failed", pr.ChecksFailed)))
					}
					if pr.ChecksPending > 0 {
						parts = append(parts, lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("● %d running", pr.ChecksPending)))
					}
					if pr.ChecksPassed > 0 {
						parts = append(parts, lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("✓ %d passed", pr.ChecksPassed)))
					}
					b.WriteString(strings.Join(parts, "  "))
					b.WriteString("\n")
				}
			}
			// pr == nil means no PR found; omit line silently
		}

		// Branch
		if selected.WorktreeBranch != "" {
			b.WriteString(wtLabelStyle.Render("Branch:  "))
			b.WriteString(wtBranchStyle.Render(selected.WorktreeBranch))
			b.WriteString("\n")
		}

		// Remote URL (lazy-cached, 5m TTL)
		h.worktreeRemoteMu.Lock()
		remoteURL, hasRemote := h.worktreeRemoteCache[selected.ID]
		_, hasRemoteTs := h.worktreeRemoteCacheTs[selected.ID]
		h.worktreeRemoteMu.Unlock()
		if !hasRemoteTs {
			b.WriteString(wtLabelStyle.Render("Remote:  "))
			b.WriteString(lipgloss.NewStyle().Foreground(ColorComment).Render("checking..."))
			b.WriteString("\n")
		} else if hasRemote && remoteURL != "" {
			displayRemote := truncatePath(remoteURL, width-4-9)
			b.WriteString(wtLabelStyle.Render("Remote:  "))
			b.WriteString(wtValueStyle.Render(displayRemote))
			b.WriteString("\n")
		}

		// Worktree path (tilde-compressed via shortenPath, already wired in Task 2)
		if selected.WorktreePath != "" {
			wtPath := shortenPath(selected.WorktreePath, width-4-9)
			b.WriteString(wtLabelStyle.Render("Path:    "))
			b.WriteString(wtValueStyle.Render(wtPath))
			b.WriteString("\n")
		}

		// Dirty status (lazy-cached, fetched via previewDebounce handler with 10s TTL)
		h.worktreeDirtyMu.Lock()
		isDirty, hasCached := h.worktreeDirtyCache[selected.ID]
		h.worktreeDirtyMu.Unlock()

		dirtyLabel := "checking..."
		dirtyStyle := wtValueStyle
		if hasCached {
			if isDirty {
				dirtyLabel = "dirty (uncommitted changes)"
				dirtyStyle = lipgloss.NewStyle().Foreground(ColorYellow)
			} else {
				dirtyLabel = "clean"
				dirtyStyle = lipgloss.NewStyle().Foreground(ColorGreen)
			}
		}
		b.WriteString(wtLabelStyle.Render("Status:  "))
		b.WriteString(dirtyStyle.Render(dirtyLabel))
		b.WriteString("\n")

		// Finish hint
		b.WriteString(wtHintStyle.Render("Finish:  "))
		b.WriteString(wtKeyStyle.Render("W"))
		b.WriteString(wtHintStyle.Render(" finish + cleanup"))
		b.WriteString("\n")
	}

	// Claude-specific info (session ID and MCPs)
	if selected.Tool == "claude" {
		// Section divider for Claude info
		claudeHeader := renderSectionDivider("Claude", width-4)
		b.WriteString(claudeHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		// Status line
		if selected.ClaudeSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			// Full session ID on its own line
			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.ClaudeSessionID))
			b.WriteString("\n")
		} else {
			statusStyle := lipgloss.NewStyle().Foreground(ColorText)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("○ Not connected"))
			b.WriteString("\n")
		}

		// MCP servers - compact format with source indicators and sync status
		mcpInfo := selected.GetMCPInfo()
		hasLoadedMCPs := len(selected.LoadedMCPNames) > 0
		hasMCPs := mcpInfo != nil && mcpInfo.HasAny()

		if hasMCPs || hasLoadedMCPs {
			b.WriteString(labelStyle.Render("MCPs:    "))

			// Build set of loaded MCPs for comparison
			loadedSet := make(map[string]bool)
			for _, name := range selected.LoadedMCPNames {
				loadedSet[name] = true
			}

			// Build set of current MCPs (from config)
			currentSet := make(map[string]bool)
			if mcpInfo != nil {
				for _, name := range mcpInfo.Global {
					currentSet[name] = true
				}
				for _, name := range mcpInfo.Project {
					currentSet[name] = true
				}
				for _, mcp := range mcpInfo.LocalMCPs {
					currentSet[mcp.Name] = true
				}
			}

			// Styles for different MCP states
			pendingStyle := lipgloss.NewStyle().Foreground(ColorYellow)
			staleStyle := lipgloss.NewStyle().Foreground(ColorText)

			var mcpParts []string

			// Helper to add MCP with appropriate styling
			addMCP := func(name, source string) {
				label := name + " (" + source + ")"
				if !hasLoadedMCPs {
					// Old session without LoadedMCPNames - show all as normal (no sync info)
					mcpParts = append(mcpParts, valueStyle.Render(label))
				} else if loadedSet[name] {
					// In both loaded and current - active (normal style)
					mcpParts = append(mcpParts, valueStyle.Render(label))
				} else {
					// In current but not loaded - pending (needs restart)
					mcpParts = append(mcpParts, pendingStyle.Render(label+" ⟳"))
				}
			}

			// Add MCPs from current config with source indicators
			if mcpInfo != nil {
				for _, name := range mcpInfo.Global {
					addMCP(name, "g")
				}
				for _, name := range mcpInfo.Project {
					addMCP(name, "p")
				}
				for _, mcp := range mcpInfo.LocalMCPs {
					// Show source path if different from project path
					sourceIndicator := "l"
					if mcp.SourcePath != selected.ProjectPath {
						// Show abbreviated path (just directory name)
						sourceIndicator = "l:" + filepath.Base(mcp.SourcePath)
					}
					addMCP(mcp.Name, sourceIndicator)
				}
			}

			// Add stale MCPs (loaded but no longer in config)
			if hasLoadedMCPs {
				for _, name := range selected.LoadedMCPNames {
					if !currentSet[name] {
						// Still running but removed from config
						mcpParts = append(mcpParts, staleStyle.Render(name+" ✕"))
					}
				}
			}

			// Calculate available width for MCPs (width - 4 for panel padding - 9 for "MCPs:    " label)
			mcpMaxWidth := width - 4 - 9
			if mcpMaxWidth < 20 {
				mcpMaxWidth = 20 // Minimum sensible width
			}

			// Build MCPs progressively to fit within available width
			var mcpResult strings.Builder
			mcpCount := 0
			currentWidth := 0

			for i, part := range mcpParts {
				// Strip ANSI codes to measure actual display width
				plainPart := tmux.StripANSI(part)
				partWidth := runewidth.StringWidth(plainPart)

				// Calculate width including separator if not first
				addedWidth := partWidth
				if mcpCount > 0 {
					addedWidth += 2 // ", " separator
				}

				remaining := len(mcpParts) - i
				isLast := remaining == 1

				// For non-last MCPs: reserve space for "+N more" indicator
				// For last MCP: just check if it fits without indicator
				var wouldExceed bool
				if isLast {
					// Last MCP - just check if it fits
					wouldExceed = currentWidth+addedWidth > mcpMaxWidth
				} else {
					// Not last - check with indicator space reserved
					moreIndicator := fmt.Sprintf(" (+%d more)", remaining)
					moreWidth := runewidth.StringWidth(moreIndicator)
					wouldExceed = currentWidth+addedWidth+moreWidth > mcpMaxWidth
				}

				if wouldExceed {
					// Would exceed - show indicator for remaining
					moreStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
					if mcpCount > 0 {
						mcpResult.WriteString(moreStyle.Render(fmt.Sprintf(" (+%d more)", remaining)))
					} else {
						// No MCPs fit - just show count
						mcpResult.WriteString(moreStyle.Render(fmt.Sprintf("(%d MCPs)", len(mcpParts))))
					}
					break
				}

				// Add separator if not first
				if mcpCount > 0 {
					mcpResult.WriteString(", ")
				}
				mcpResult.WriteString(part)
				currentWidth += addedWidth
				mcpCount++
			}

			b.WriteString(mcpResult.String())
			b.WriteString("\n")
		}

		// Fork hint when session can be forked
		if selected.CanFork() {
			hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
			keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			b.WriteString(hintStyle.Render("Fork:    "))
			b.WriteString(keyStyle.Render("f"))
			b.WriteString(hintStyle.Render(" quick fork, "))
			b.WriteString(keyStyle.Render("F"))
			b.WriteString(hintStyle.Render(" fork with options"))
			b.WriteString("\n")
		}
	}

	// Gemini-specific info (session ID)
	if selected.Tool == "gemini" {
		geminiHeader := renderSectionDivider("Gemini", width-4)
		b.WriteString(geminiHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		if selected.GeminiSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.GeminiSessionID))
			b.WriteString("\n")

			// Display active model
			modelDisplay := "auto"
			if selected.GeminiModel != "" {
				modelDisplay = selected.GeminiModel
			}
			accentStyle := lipgloss.NewStyle().Foreground(ColorAccent)
			b.WriteString(labelStyle.Render("Model:   "))
			b.WriteString(accentStyle.Render(modelDisplay))
			b.WriteString("\n")

			// MCPs for Gemini (global only)
			mcpInfo := selected.GetMCPInfo()
			renderSimpleMCPLine(&b, mcpInfo, width)
		} else {
			statusStyle := lipgloss.NewStyle().Foreground(ColorText)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("○ Not connected"))
			b.WriteString("\n")
		}
	}

	// OpenCode-specific info (session ID)
	if selected.Tool == "opencode" {
		opencodeHeader := renderSectionDivider("OpenCode", width-4)
		b.WriteString(opencodeHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		// Debug: log what value we're seeing
		uiLog.Debug("opencode_rendering_preview", slog.String("title", selected.Title), slog.String("session_id", selected.OpenCodeSessionID))

		if selected.OpenCodeSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.OpenCodeSessionID))
			b.WriteString("\n")

			// Show when session was detected
			if !selected.OpenCodeDetectedAt.IsZero() {
				detectedAgo := formatRelativeTime(selected.OpenCodeDetectedAt)
				dimStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
				b.WriteString(labelStyle.Render("Detected:"))
				b.WriteString(dimStyle.Render(" " + detectedAgo))
				b.WriteString("\n")
			}

			// Fork hint for OpenCode
			if selected.CanFork() {
				renderForkHintLine(&b)
			}
		} else {
			// Check if detection has completed (OpenCodeDetectedAt is set even when no session found)
			if selected.OpenCodeDetectedAt.IsZero() {
				// Detection not yet completed - show detecting state
				statusStyle := lipgloss.NewStyle().Foreground(ColorYellow)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("◐ Detecting session..."))
				b.WriteString("\n")
			} else {
				// Detection completed but no session found
				statusStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("○ No session found"))
				b.WriteString("\n")
			}
		}
	}

	// Codex-specific info (session ID, detection)
	if selected.Tool == "codex" {
		codexHeader := renderSectionDivider("Codex", width-4)
		b.WriteString(codexHeader)
		b.WriteString("\n")

		renderToolStatusLine(&b, selected.CodexSessionID, selected.CodexDetectedAt, true)
		if selected.CodexSessionID != "" {
			renderDetectedAtLine(&b, selected.CodexDetectedAt)
		}
	}

	// Custom tool info (tools defined in config.toml that aren't built-in)
	if selected.Tool != "claude" && selected.Tool != "gemini" && selected.Tool != "opencode" && selected.Tool != "codex" {
		if toolDef := session.GetToolDef(selected.Tool); toolDef != nil {
			toolName := selected.Tool
			if toolDef.Icon != "" {
				toolName = toolDef.Icon + " " + toolName
			}
			customHeader := renderSectionDivider(toolName, width-4)
			b.WriteString(customHeader)
			b.WriteString("\n")

			labelStyle := lipgloss.NewStyle().Foreground(ColorText)

			genericID := selected.GetGenericSessionID()
			if genericID != "" {
				statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
				valueStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("● Connected"))
				b.WriteString("\n")

				b.WriteString(labelStyle.Render("Session: "))
				b.WriteString(valueStyle.Render(genericID))
				b.WriteString("\n")
			} else {
				statusStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("○ Not connected"))
				b.WriteString("\n")
			}

			// Resume hint when tool supports restart with session resume
			if selected.CanRestartGeneric() {
				hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
				keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
				b.WriteString(hintStyle.Render("Resume:  "))
				b.WriteString(keyStyle.Render("r"))
				b.WriteString(hintStyle.Render(" restart with session resume"))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	// Special handling for error state - show guidance instead of output
	if selected.Status == session.StatusError {
		errorHeader := renderSectionDivider("Session Inactive", width-4)
		b.WriteString(errorHeader)
		b.WriteString("\n\n")

		// Warning icon and message
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		dimStyle := lipgloss.NewStyle().Foreground(ColorText)
		keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

		b.WriteString(warnStyle.Render("⚠ No tmux session running"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("This can happen if:"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • Session was added but not yet started"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • tmux server was restarted"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • Terminal was closed or system rebooted"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Actions:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("R"))
		b.WriteString(dimStyle.Render(" Start   - create and start tmux session"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("d"))
		b.WriteString(dimStyle.Render(" Delete  - remove from list"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("Enter"))
		b.WriteString(dimStyle.Render(" - attach (will auto-start)"))
		b.WriteString("\n")

		// Pad output to exact height to prevent layout shifts
		content := b.String()
		lines := strings.Split(content, "\n")
		lineCount := len(lines)

		if lineCount < height {
			for i := lineCount; i < height; i++ {
				content += "\n"
			}
		}

		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}

		return content
	}

	// Check preview settings for what to show
	config, _ := session.LoadUserConfig()
	showOutput := config == nil || config.GetShowOutput() // Default to true if config fails

	// Check if session is launching/resuming (for animation priority)
	_, isSessionLaunching := h.launchingSessions[selected.ID]
	_, isSessionResuming := h.resumingSessions[selected.ID]
	_, isSessionForking := h.forkingSessions[selected.ID]
	isStartingUp := isSessionLaunching || isSessionResuming || isSessionForking

	// If output is disabled AND not starting up, return early
	// (We want to show the launch animation even if output is normally disabled)
	if !showOutput && !isStartingUp {
		infoCard := h.renderSessionInfoCard(selected, width, height)
		b.WriteString("\n")
		b.WriteString(infoCard)

		// Pad output to exact height to prevent layout shifts
		content := b.String()
		lines := strings.Split(content, "\n")
		lineCount := len(lines)
		if lineCount < height {
			for i := lineCount; i < height; i++ {
				content += "\n"
			}
		}
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}
		return content
	}

	// Terminal output header
	termHeader := renderSectionDivider("Output", width-4)
	b.WriteString(termHeader)
	b.WriteString("\n")

	// Render diffstat line using the dedicated method (avoids fragile ANSI scanning).
	if line := h.preview.DiffStatLine(); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Check if this session is launching (newly created), resuming (restarted), or forking
	launchTime, isLaunching := h.launchingSessions[selected.ID]
	resumeTime, isResuming := h.resumingSessions[selected.ID]
	forkTime, isForking := h.forkingSessions[selected.ID]

	// Determine if we should show animation (launch, resume, MCP loading, or forking)
	// For Claude: show for minimum 6 seconds, then check for ready indicators
	// For others: show for first 3 seconds after creation
	showLaunchingAnimation := false
	showForkingAnimation := isForking // Show forking animation immediately
	var animationStartTime time.Time
	if isLaunching {
		animationStartTime = launchTime
	} else if isResuming {
		animationStartTime = resumeTime
	}

	// Apply STATUS-BASED animation logic (matches hasActiveAnimation exactly)
	// Animation shows until session is ready, detected via status or content
	if isLaunching || isResuming {
		timeSinceStart := time.Since(animationStartTime)

		// Brief minimum to prevent flicker
		if timeSinceStart < launchAnimationMinDuration(selected.Tool) {
			showLaunchingAnimation = true
		} else if timeSinceStart < 15*time.Second {
			// STATUS-BASED CHECK: Session ready when Running/Waiting/Idle
			sessionReady := selected.Status == session.StatusRunning ||
				selected.Status == session.StatusWaiting ||
				selected.Status == session.StatusIdle

			if !sessionReady {
				// Also check content for faster detection
				h.previewCacheMu.RLock()
				previewContent := h.previewCache[selected.ID]
				h.previewCacheMu.RUnlock()

				if selected.Tool == "claude" || selected.Tool == "gemini" {
					// Claude/Gemini ready indicators
					agentReady := strings.Contains(previewContent, "ctrl+c to interrupt") ||
						strings.Contains(previewContent, "No, and tell Claude what to do differently") ||
						strings.Contains(previewContent, "\n> ") ||
						strings.Contains(previewContent, "> \n") ||
						strings.Contains(previewContent, "esc to interrupt") ||
						strings.Contains(previewContent, "⠋") || strings.Contains(previewContent, "⠙") ||
						strings.Contains(previewContent, "Thinking") ||
						strings.Contains(previewContent, "╭─")

					if selected.Tool == "gemini" {
						agentReady = agentReady ||
							strings.Contains(previewContent, "▸") ||
							strings.Contains(previewContent, "gemini>")
					}

					if !agentReady {
						showLaunchingAnimation = true
					}
				} else {
					// Non-Claude/Gemini: ready if substantial content
					if len(strings.TrimSpace(previewContent)) <= 50 {
						showLaunchingAnimation = true
					}
				}
			}
		}
		// After 15 seconds, animation stops regardless
	}

	// Terminal preview - use cached content (async fetching keeps View() pure)
	h.previewCacheMu.RLock()
	preview, hasCached := h.previewCache[selected.ID]
	h.previewCacheMu.RUnlock()

	// Show forking animation when fork is in progress (highest priority)
	if showForkingAnimation {
		b.WriteString("\n")
		b.WriteString(h.renderForkingState(selected, width, forkTime))
	} else if showLaunchingAnimation {
		// Show launching animation for new sessions
		b.WriteString("\n")
		b.WriteString(h.renderLaunchingState(selected, width, animationStartTime))
	} else if !hasCached {
		// Show loading indicator while waiting for async fetch
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorText).
			Italic(true)
		b.WriteString(loadingStyle.Render("Loading preview..."))
	} else if preview == "" {
		emptyTerm := lipgloss.NewStyle().
			Foreground(ColorText).
			Italic(true).
			Render("(terminal is empty)")
		b.WriteString(emptyTerm)
	} else {
		// Calculate maxLines dynamically based on how many header lines we've already written
		// This accounts for Claude sessions having more header lines than other sessions
		currentContent := b.String()
		headerLines := strings.Count(currentContent, "\n") + 1 // +1 for the current line
		lines := strings.Split(preview, "\n")

		// Strip trailing empty lines BEFORE truncation
		// This ensures we show actual content, not empty trailing lines when space is limited
		// (Terminal output often ends with empty lines at cursor position)
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}

		// If all lines were empty, show empty indicator
		if len(lines) == 0 {
			emptyTerm := lipgloss.NewStyle().
				Foreground(ColorText).
				Italic(true).
				Render("(terminal is empty)")
			b.WriteString(emptyTerm)
			return b.String()
		}

		maxLines := height - headerLines - 1 // -1 for potential truncation indicator
		if maxLines < 1 {
			maxLines = 1
		}

		// Track if we're truncating from the top (for indicator)
		truncatedFromTop := len(lines) > maxLines
		truncatedCount := 0
		if truncatedFromTop {
			// Reserve one line for the truncation indicator
			maxLines--
			if maxLines < 1 {
				maxLines = 1
			}
			truncatedCount = len(lines) - maxLines
			lines = lines[len(lines)-maxLines:]
		}

		maxWidth := width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}

		// Show truncation indicator if content was cut from top
		if truncatedFromTop {
			truncIndicator := lipgloss.NewStyle().
				Foreground(ColorText).
				Italic(true).
				Render(fmt.Sprintf("⋮ %d more lines above", truncatedCount))
			b.WriteString(truncIndicator)
			b.WriteString("\n")
		}

		// Track consecutive empty lines to preserve some spacing
		consecutiveEmpty := 0
		const maxConsecutiveEmpty = 2 // Allow up to 2 consecutive empty lines

		for _, line := range lines {
			// Plain version (no ANSI, no control chars) used only for measurement and
			// empty-line detection. renderLine keeps ANSI escape sequences for display.
			plainLine := stripControlChars(tmux.StripANSI(line))
			renderLine := stripControlCharsPreserveANSI(line)

			// Handle empty lines - preserve some for readability
			trimmed := strings.TrimSpace(plainLine)
			if trimmed == "" {
				consecutiveEmpty++
				if consecutiveEmpty <= maxConsecutiveEmpty {
					b.WriteString("\n") // Preserve empty line
				}
				continue
			}
			consecutiveEmpty = 0 // Reset counter on non-empty line

			// Truncate based on display width (handles CJK, emoji correctly).
			// Fall back to plain when truncating — can't safely split mid-escape.
			displayWidth := runewidth.StringWidth(plainLine)
			if displayWidth > maxWidth {
				renderLine = runewidth.Truncate(plainLine, maxWidth-3, "...")
			}

			// Don't wrap with previewStyle.Render — tmux ANSI sequences handle styling.
			b.WriteString(renderLine)
			b.WriteString("\n")
		}
	}

	// CRITICAL: Enforce width constraint on ALL lines to prevent overflow into left panel
	// When lipgloss.JoinHorizontal combines panels, any line exceeding rightWidth
	// will wrap and corrupt the layout
	maxWidth := width - 2 // Small margin for safety
	if maxWidth < 20 {
		maxWidth = 20
	}

	result := b.String()
	lines := strings.Split(result, "\n")
	var truncatedLines []string
	for _, line := range lines {
		// Strip ANSI codes for accurate measurement
		cleanLine := tmux.StripANSI(line)
		displayWidth := runewidth.StringWidth(cleanLine)
		if displayWidth > maxWidth {
			// Truncate the clean version, then re-apply basic styling
			// Note: This loses original styling but prevents layout corruption
			truncated := runewidth.Truncate(cleanLine, maxWidth-3, "...")
			truncatedLines = append(truncatedLines, truncated)
		} else {
			truncatedLines = append(truncatedLines, line)
		}
	}

	return strings.Join(truncatedLines, "\n")
}

// stripControlChars removes C0 control characters (except \n and \t) from a string.
// tmux capture-pane output may include \r, \b, and other control characters that
// corrupt terminal rendering when embedded inside styled TUI output (e.g. \r moves
// the cursor to column 0, overwriting the left panel in a JoinHorizontal layout).
func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' {
			return -1 // Drop the character
		}
		return r
	}, s)
}

// stripControlCharsPreserveANSI removes C0 control chars but keeps ESC (0x1b)
// so that ANSI escape sequences captured via tmux capture-pane -e pass through.
func stripControlCharsPreserveANSI(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' && r != 0x1b {
			return -1
		}
		return r
	}, s)
}

// truncatePath shortens a path to fit within maxLen display width
func truncatePath(path string, maxLen int) string {
	pathWidth := runewidth.StringWidth(path)
	if pathWidth <= maxLen {
		return path
	}
	if maxLen < 10 {
		maxLen = 10
	}
	// Show beginning and end: /Users/.../project
	// Use rune-based slicing for proper Unicode handling
	runes := []rune(path)
	startLen := maxLen / 3
	endLen := maxLen*2/3 - 3
	if startLen+endLen+3 > len(runes) {
		// Path is short in runes but wide in display - use simple truncation
		return runewidth.Truncate(path, maxLen-3, "...")
	}
	return string(runes[:startLen]) + "..." + string(runes[len(runes)-endLen:])
}

// shortenPath replaces the home directory prefix with ~ then truncates to fit maxLen.
func shortenPath(path string, maxLen int) string {
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(path, home+"/") {
			path = "~/" + path[len(home)+1:]
		} else if path == home {
			path = "~"
		}
	}
	return truncatePath(path, maxLen)
}

// formatRelativeTime formats a time as a human-readable relative string
// Examples: "just now", "2m ago", "1h ago", "3h ago", "1d ago"
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// renderGroupPreview renders the preview pane for a group
func (h *Home) renderGroupPreview(group *session.Group, width, height int) string {
	var b strings.Builder

	// Group header with folder icon
	headerStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	b.WriteString(headerStyle.Render("📁 " + group.Name))
	b.WriteString("\n\n")

	// Session count
	countStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)
	b.WriteString(countStyle.Render(fmt.Sprintf("%d sessions", len(group.Sessions))))
	b.WriteString("\n\n")

	// Status breakdown with inline badges
	running, waiting, idle, errored := 0, 0, 0, 0
	for _, sess := range group.Sessions {
		switch sess.Status {
		case session.StatusRunning:
			running++
		case session.StatusWaiting:
			waiting++
		case session.StatusIdle:
			idle++
		case session.StatusError:
			errored++
		}
	}

	// Compact status line (inline, not badges)
	var statuses []string
	if running > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("● %d running", running)))
	}
	if waiting > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("◐ %d waiting", waiting)))
	}
	if idle > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("○ %d idle", idle)))
	}
	if errored > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✕ %d error", errored)))
	}

	if len(statuses) > 0 {
		b.WriteString(strings.Join(statuses, "  "))
		b.WriteString("\n\n")
	}

	// Repository worktree summary (when all sessions share the same repo root)
	if repoInfo := h.getGroupWorktreeInfo(group); repoInfo != nil {
		b.WriteString(renderSectionDivider("Repository", width-4))
		b.WriteString("\n")

		repoLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
		repoValueStyle := lipgloss.NewStyle().Foreground(ColorText)
		repoBranchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

		b.WriteString(repoLabelStyle.Render("Repo:       "))
		b.WriteString(repoValueStyle.Render(truncatePath(repoInfo.repoRoot, width-4-12)))
		b.WriteString("\n")

		b.WriteString(repoLabelStyle.Render("Worktrees:  "))
		b.WriteString(repoValueStyle.Render(fmt.Sprintf("%d active", len(repoInfo.branches))))
		b.WriteString("\n")

		for _, br := range repoInfo.branches {
			dirtyMark := ""
			if br.dirtyChecked {
				if br.isDirty {
					dirtyMark = lipgloss.NewStyle().Foreground(ColorYellow).Render(" (dirty)")
				} else {
					dirtyMark = lipgloss.NewStyle().Foreground(ColorGreen).Render(" (clean)")
				}
			}
			b.WriteString("  ")
			b.WriteString(repoBranchStyle.Render("• " + br.branch))
			b.WriteString(dirtyMark)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Todos section — always shown when project has a path
	if projectPath := h.getDefaultPathForGroup(group.Path); projectPath != "" {
		todos, err := h.storage.LoadTodos(projectPath)
		if err != nil {
			uiLog.Debug("renderGroupPreview: load todos", slog.String("path", projectPath), slog.String("err", err.Error()))
		}
		b.WriteString(renderSectionDivider(fmt.Sprintf("Todos (%d)", len(todos)), width-4))
		b.WriteString("\n")
		if len(todos) == 0 {
			b.WriteString(DimStyle.Render("  No todos — press t to add"))
			b.WriteString("\n")
		} else {
			maxTodos := 6
			for i, t := range todos {
				if i >= maxTodos {
					b.WriteString(DimStyle.Render(fmt.Sprintf("  ... +%d more", len(todos)-i)))
					b.WriteString("\n")
					break
				}
				st := todoStatusStyle(t.Status)
				icon := st.Render(todoStatusIcon(t.Status))
				title := t.Title
				maxTitleWidth := width - 8
				if maxTitleWidth > 0 && len(title) > maxTitleWidth {
					title = title[:maxTitleWidth-3] + "..."
				}
				b.WriteString(fmt.Sprintf("  %s %s\n", icon, lipgloss.NewStyle().Foreground(ColorText).Render(title)))
			}
		}
		b.WriteString("\n")
	}

	// Sessions divider
	b.WriteString(renderSectionDivider("Sessions", width-4))
	b.WriteString("\n")

	// Session list (compact)
	if len(group.Sessions) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
		b.WriteString(emptyStyle.Render("  No sessions in this project"))
		b.WriteString("\n")
	} else {
		maxShow := height - 12
		if maxShow < 3 {
			maxShow = 3
		}
		for i, sess := range group.Sessions {
			if i >= maxShow {
				remaining := len(group.Sessions) - i
				b.WriteString(DimStyle.Render(fmt.Sprintf("  ... +%d more", remaining)))
				break
			}

			// Status icon
			statusIcon := "○"
			statusColor := ColorTextDim
			switch sess.Status {
			case session.StatusRunning:
				statusIcon, statusColor = "●", ColorGreen
			case session.StatusWaiting:
				statusIcon, statusColor = "◐", ColorYellow
			case session.StatusError:
				statusIcon, statusColor = "✕", ColorRed
			}
			status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
			name := lipgloss.NewStyle().Foreground(ColorText).Render(sess.Title)
			tool := lipgloss.NewStyle().Foreground(ColorPurple).Faint(true).Render(sess.Tool)

			b.WriteString(fmt.Sprintf("  %s %s %s\n", status, name, tool))
		}
	}

	// Keyboard hints at bottom
	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
	b.WriteString(hintStyle.Render("t todos • n new session • R rename • d delete"))

	// CRITICAL: Enforce width constraint on ALL lines to prevent overflow into left panel
	maxWidth := width - 2
	if maxWidth < 20 {
		maxWidth = 20
	}

	result := b.String()
	lines := strings.Split(result, "\n")
	var truncatedLines []string
	for _, line := range lines {
		cleanLine := tmux.StripANSI(line)
		displayWidth := runewidth.StringWidth(cleanLine)
		if displayWidth > maxWidth {
			truncated := runewidth.Truncate(cleanLine, maxWidth-3, "...")
			truncatedLines = append(truncatedLines, truncated)
		} else {
			truncatedLines = append(truncatedLines, line)
		}
	}

	return strings.Join(truncatedLines, "\n")
}

// groupWorktreeBranch holds info about a single worktree branch in a group
type groupWorktreeBranch struct {
	branch       string
	isDirty      bool
	dirtyChecked bool
}

// groupWorktreeInfo holds aggregated worktree info for a group sharing a common repo
type groupWorktreeInfo struct {
	repoRoot string
	branches []groupWorktreeBranch
}

// getGroupWorktreeInfo returns worktree summary if all sessions in the group
// share the same repo root and at least one is a worktree. Returns nil otherwise.
func (h *Home) getGroupWorktreeInfo(group *session.Group) *groupWorktreeInfo {
	if len(group.Sessions) < 2 {
		return nil
	}

	// Check if all sessions share a common repo root and count worktrees
	var commonRepo string
	var branches []groupWorktreeBranch
	for _, sess := range group.Sessions {
		if !sess.IsWorktree() {
			continue
		}
		if commonRepo == "" {
			commonRepo = sess.WorktreeRepoRoot
		} else if sess.WorktreeRepoRoot != commonRepo {
			return nil // Different repos, skip
		}

		// Get dirty status from cache
		h.worktreeDirtyMu.Lock()
		isDirty, hasCached := h.worktreeDirtyCache[sess.ID]
		h.worktreeDirtyMu.Unlock()

		branches = append(branches, groupWorktreeBranch{
			branch:       sess.WorktreeBranch,
			isDirty:      isDirty,
			dirtyChecked: hasCached,
		})
	}

	if len(branches) == 0 {
		return nil
	}

	return &groupWorktreeInfo{
		repoRoot: commonRepo,
		branches: branches,
	}
}
