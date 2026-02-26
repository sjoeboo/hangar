package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmType indicates what action is being confirmed
type ConfirmType int

const (
	ConfirmDeleteSession ConfirmType = iota
	ConfirmDeleteGroup
	ConfirmQuitWithPool
	ConfirmCreateDirectory
	ConfirmInstallHooks
	ConfirmBulkDeleteSessions
	ConfirmBulkRestart
)

// ConfirmDialog handles confirmation for destructive actions
type ConfirmDialog struct {
	visible     bool
	confirmType ConfirmType
	targetID    string // Session ID or group path
	targetName  string // Display name
	width       int
	height      int
	mcpCount    int // Number of running MCPs (for quit confirmation)

	// Bulk operation fields
	targetIDs   []string // Session IDs for bulk operations
	targetNames []string // Display names for bulk targets

	// Pending session creation data (for ConfirmCreateDirectory)
	pendingSessionName      string
	pendingSessionPath      string
	pendingSessionCommand   string
	pendingSessionGroupPath string
	pendingToolOptionsJSON  json.RawMessage // Generic tool options (claude, codex, etc.)
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog() *ConfirmDialog {
	return &ConfirmDialog{}
}

// ShowDeleteSession shows confirmation for session deletion
func (c *ConfirmDialog) ShowDeleteSession(sessionID, sessionName string) {
	c.visible = true
	c.confirmType = ConfirmDeleteSession
	c.targetID = sessionID
	c.targetName = sessionName
}

// ShowDeleteGroup shows confirmation for group deletion
func (c *ConfirmDialog) ShowDeleteGroup(groupPath, groupName string) {
	c.visible = true
	c.confirmType = ConfirmDeleteGroup
	c.targetID = groupPath
	c.targetName = groupName
}

// ShowQuitWithPool shows confirmation for quitting with MCP pool running
func (c *ConfirmDialog) ShowQuitWithPool(mcpCount int) {
	c.visible = true
	c.confirmType = ConfirmQuitWithPool
	c.mcpCount = mcpCount
	c.targetID = ""
	c.targetName = ""
}

// ShowCreateDirectory shows confirmation for creating a missing directory
func (c *ConfirmDialog) ShowCreateDirectory(path, sessionName, command, groupPath string, toolOptionsJSON json.RawMessage) {
	c.visible = true
	c.confirmType = ConfirmCreateDirectory
	c.targetID = path
	c.targetName = path
	c.pendingSessionName = sessionName
	c.pendingSessionPath = path
	c.pendingSessionCommand = command
	c.pendingSessionGroupPath = groupPath
	c.pendingToolOptionsJSON = toolOptionsJSON
}

// ShowInstallHooks shows confirmation for installing Claude Code hooks
func (c *ConfirmDialog) ShowInstallHooks() {
	c.visible = true
	c.confirmType = ConfirmInstallHooks
	c.targetID = ""
	c.targetName = ""
}

// ShowBulkDeleteSessions shows confirmation for bulk session deletion
func (c *ConfirmDialog) ShowBulkDeleteSessions(ids []string, names []string) {
	c.visible = true
	c.confirmType = ConfirmBulkDeleteSessions
	c.targetIDs = ids
	c.targetNames = names
	c.targetID = ""
	c.targetName = fmt.Sprintf("%d sessions", len(ids))
}

// ShowBulkRestart shows confirmation for bulk session restart
func (c *ConfirmDialog) ShowBulkRestart(count int) {
	c.visible = true
	c.confirmType = ConfirmBulkRestart
	c.targetID = ""
	c.targetName = fmt.Sprintf("%d sessions", count)
}

// GetTargetIDs returns the session IDs for bulk operations
func (c *ConfirmDialog) GetTargetIDs() []string {
	return c.targetIDs
}

// GetPendingSession returns the pending session creation data
func (c *ConfirmDialog) GetPendingSession() (name, path, command, groupPath string, toolOptionsJSON json.RawMessage) {
	return c.pendingSessionName, c.pendingSessionPath, c.pendingSessionCommand, c.pendingSessionGroupPath, c.pendingToolOptionsJSON
}

// Hide hides the dialog
func (c *ConfirmDialog) Hide() {
	c.visible = false
	c.targetID = ""
	c.targetName = ""
	c.targetIDs = nil
	c.targetNames = nil
}

// IsVisible returns whether the dialog is visible
func (c *ConfirmDialog) IsVisible() bool {
	return c.visible
}

// GetTargetID returns the session ID or group path being confirmed
func (c *ConfirmDialog) GetTargetID() string {
	return c.targetID
}

// GetConfirmType returns the type of confirmation
func (c *ConfirmDialog) GetConfirmType() ConfirmType {
	return c.confirmType
}

// SetSize updates dialog dimensions
func (c *ConfirmDialog) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Update handles key events
func (c *ConfirmDialog) Update(msg tea.KeyMsg) (*ConfirmDialog, tea.Cmd) {
	// Dialog handles y/n/enter/esc only
	return c, nil
}

// View renders the confirmation dialog
func (c *ConfirmDialog) View() string {
	if !c.visible {
		return ""
	}

	// Build warning message and buttons based on action type
	var title, warning, details string
	var buttons string
	var borderColor lipgloss.Color

	// Styles (shared)
	detailsStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		MarginBottom(1)

	switch c.confirmType {
	case ConfirmDeleteSession:
		title = "⚠️  Delete Session?"
		warning = fmt.Sprintf("This will PERMANENTLY KILL the tmux session:\n\n  \"%s\"", c.targetName)
		details = "• The tmux session will be terminated\n• Any running processes will be killed\n• Terminal history will be lost\n• Press Ctrl+Z after deletion to undo"
		borderColor = ColorRed

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 2).
			Bold(true).
			Render("y Delete")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Padding(0, 2).
			Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmDeleteGroup:
		title = "⚠️  Delete Group?"
		warning = fmt.Sprintf("This will delete the group:\n\n  \"%s\"", c.targetName)
		details = "• All sessions will be MOVED to 'default' group\n• Sessions will NOT be killed\n• The group structure will be lost"
		borderColor = ColorRed

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 2).
			Bold(true).
			Render("y Delete")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Padding(0, 2).
			Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmQuitWithPool:
		title = "MCP Pool Running"
		warning = fmt.Sprintf("%d MCP servers are running in the pool.", c.mcpCount)
		details = "Keep them running for faster startup next time,\nor shut down to free resources."
		borderColor = ColorAccent

		// "Keep running" is the default (green), "Shut down" is secondary (red)
		buttonKeep := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Padding(0, 2).
			Bold(true).
			Render("k Keep running")
		buttonShutdown := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 2).
			Bold(true).
			Render("s Shut down")
		escHint := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonKeep, "  ", buttonShutdown, "  ", escHint)

	case ConfirmCreateDirectory:
		title = "📁  Directory Not Found"
		warning = fmt.Sprintf("The path does not exist:\n\n  %s", c.targetName)
		details = "Create this directory and start the session?"
		borderColor = ColorAccent

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Padding(0, 2).
			Bold(true).
			Render("y Create")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorRed).
			Padding(0, 2).
			Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmInstallHooks:
		title = "Claude Code Hooks"
		warning = "Hangar can install Claude Code lifecycle hooks\nfor real-time status detection (instant green/yellow/gray)."
		details = "This writes to your Claude settings.json (preserves existing settings).\nNew/restarted sessions will use hooks; existing sessions continue unchanged.\nYou can disable later with: hooks_enabled = false in config.toml"
		borderColor = ColorAccent

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Padding(0, 2).
			Bold(true).
			Render("y Install")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Padding(0, 2).
			Bold(true).
			Render("n Skip")
		escHint := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(Esc to skip)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmBulkDeleteSessions:
		title = fmt.Sprintf("⚠️  Delete %s?", c.targetName)
		// Build session list (show up to 8, then "and N more...")
		shown := c.targetNames
		extra := 0
		if len(c.targetNames) > 8 {
			shown = c.targetNames[:8]
			extra = len(c.targetNames) - 8
		}
		listStr := ""
		for _, name := range shown {
			listStr += fmt.Sprintf("  • %s\n", name)
		}
		if extra > 0 {
			listStr += fmt.Sprintf("  … and %d more", extra)
		}
		warning = "This will PERMANENTLY KILL these tmux sessions:\n\n" + strings.TrimRight(listStr, "\n")
		details = "• Worktrees will be removed where applicable\n• Press Ctrl+Z after deletion to undo (one at a time)"
		borderColor = ColorRed

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorRed).Padding(0, 2).Bold(true).
			Render(fmt.Sprintf("y Delete %s", c.targetName))
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorAccent).Padding(0, 2).Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().Foreground(ColorTextDim).Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)

	case ConfirmBulkRestart:
		title = fmt.Sprintf("Restart %s?", c.targetName)
		warning = fmt.Sprintf("Restart all %s?", c.targetName)
		details = "Each session will be recreated and resumed."
		borderColor = ColorAccent

		buttonYes := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorAccent).Padding(0, 2).Bold(true).
			Render("y Restart")
		buttonNo := lipgloss.NewStyle().
			Foreground(ColorBg).Background(ColorRed).Padding(0, 2).Bold(true).
			Render("n Cancel")
		escHint := lipgloss.NewStyle().Foreground(ColorTextDim).Render("(Esc to cancel)")
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, buttonYes, "  ", buttonNo, "  ", escHint)
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(borderColor).
		MarginBottom(1)

	// Warning style
	warningStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		MarginBottom(1)

	// Build content
	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		warningStyle.Render(warning),
		detailsStyle.Render(details),
		"",
		buttons,
	)

	// Dialog box
	dialogWidth := 50
	if c.width > 0 && c.width < dialogWidth+10 {
		dialogWidth = c.width - 10
	}

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)

	// Center in screen
	if c.width > 0 && c.height > 0 {
		// Create full-screen overlay with centered dialog
		dialogHeight := lipgloss.Height(dialogBox)
		dialogWidth := lipgloss.Width(dialogBox)

		padLeft := (c.width - dialogWidth) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		padTop := (c.height - dialogHeight) / 2
		if padTop < 0 {
			padTop = 0
		}

		// Build centered dialog
		var b strings.Builder
		for i := 0; i < padTop; i++ {
			b.WriteString("\n")
		}
		for _, line := range strings.Split(dialogBox, "\n") {
			b.WriteString(strings.Repeat(" ", padLeft))
			b.WriteString(line)
			b.WriteString("\n")
		}

		return b.String()
	}

	return dialogBox
}
