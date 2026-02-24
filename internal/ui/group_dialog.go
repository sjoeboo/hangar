package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GroupDialogMode represents the dialog mode
type GroupDialogMode int

const (
	GroupDialogCreate GroupDialogMode = iota
	GroupDialogRename
	GroupDialogMove
	GroupDialogRenameSession
)

// GroupDialog handles group/project creation, renaming, and moving sessions
type GroupDialog struct {
	visible    bool
	mode       GroupDialogMode
	nameInput  textinput.Model
	pathInput  textinput.Model // only used in Create mode for root-level projects
	focusIndex int             // 0=name, 1=path (Create root only)
	width      int
	height     int
	groupPath  string   // Current group being edited (for rename) or parent path (for create subgroup)
	parentName string   // Display name of parent group (for subgroup creation)
	groupNames []string // Available groups (for move)
	selected   int      // Selected group index (for move)
	sessionID  string   // Session ID being renamed (for rename session)

	validationErr string // Inline validation error displayed inside the dialog

	// Tab toggle between Root and Subgroup modes (Issue #111)
	contextParentPath string // Original cursor context parent path (for toggling back)
	contextParentName string // Original cursor context parent name (for toggling back)
}

// NewGroupDialog creates a new group dialog
func NewGroupDialog() *GroupDialog {
	ti := textinput.New()
	ti.Placeholder = "Project name"
	ti.CharLimit = 50
	ti.Width = 30

	pi := textinput.New()
	pi.Placeholder = "~/path/to/repo"
	pi.CharLimit = 256
	pi.Width = 30

	return &GroupDialog{
		nameInput:  ti,
		pathInput:  pi,
		groupNames: []string{},
	}
}

// isRootCreate returns true when we're creating a root-level project (path input shown).
func (g *GroupDialog) isRootCreate() bool {
	return g.mode == GroupDialogCreate && g.groupPath == ""
}

// Show shows the dialog in create mode (root level group/project)
func (g *GroupDialog) Show() {
	g.visible = true
	g.mode = GroupDialogCreate
	g.groupPath = "" // No parent = root level
	g.parentName = ""
	g.focusIndex = 0
	g.validationErr = ""
	g.nameInput.SetValue("")
	g.nameInput.Focus()
	g.pathInput.Blur()
	g.initPathDefault()
}

// ShowCreateSubgroup shows the dialog for creating a subgroup under a parent
func (g *GroupDialog) ShowCreateSubgroup(parentPath, parentName string) {
	g.visible = true
	g.mode = GroupDialogCreate
	g.groupPath = parentPath // Parent path for the new subgroup
	g.parentName = parentName
	g.focusIndex = 0
	g.validationErr = ""
	g.nameInput.SetValue("")
	g.nameInput.Focus()
	g.pathInput.Blur()
}

// ShowCreateWithContext opens the create dialog with cursor context for Tab toggling.
// If parentPath is non-empty, defaults to subgroup mode with Tab toggle available.
// If parentPath is empty, opens as root-level group with no toggle.
func (g *GroupDialog) ShowCreateWithContext(parentPath, parentName string) {
	g.visible = true
	g.mode = GroupDialogCreate
	g.contextParentPath = parentPath
	g.contextParentName = parentName
	g.focusIndex = 0
	g.validationErr = ""
	g.nameInput.SetValue("")
	g.nameInput.Focus()
	g.pathInput.Blur()

	if parentPath != "" {
		// Default to subgroup mode
		g.groupPath = parentPath
		g.parentName = parentName
	} else {
		// Root mode
		g.groupPath = ""
		g.parentName = ""
		g.initPathDefault()
	}
}

// ShowCreateWithContextDefaultRoot opens the create dialog defaulting to root mode,
// but stores the cursor context so Tab toggle can switch to subgroup mode.
// Used when the cursor is on a session inside a group (not on the group header itself).
func (g *GroupDialog) ShowCreateWithContextDefaultRoot(parentPath, parentName string) {
	g.visible = true
	g.mode = GroupDialogCreate
	g.contextParentPath = parentPath
	g.contextParentName = parentName
	g.focusIndex = 0
	g.validationErr = ""
	g.nameInput.SetValue("")
	g.nameInput.Focus()
	g.pathInput.Blur()

	// Default to root mode, Tab toggles to subgroup
	g.groupPath = ""
	g.parentName = ""
	g.initPathDefault()
}

// initPathDefault pre-populates the path input with the current working directory.
func (g *GroupDialog) initPathDefault() {
	if cwd, err := os.Getwd(); err == nil {
		g.pathInput.SetValue(cwd)
	} else {
		g.pathInput.SetValue("")
	}
}

// CanToggle returns true when the Tab toggle between Root and Subgroup is available.
// Only applies in Create mode when the cursor was on a group context.
func (g *GroupDialog) CanToggle() bool {
	return g.mode == GroupDialogCreate && g.contextParentPath != ""
}

// ToggleRootSubgroup swaps between root-level and subgroup creation modes.
func (g *GroupDialog) ToggleRootSubgroup() {
	if !g.CanToggle() {
		return
	}
	if g.groupPath == "" {
		// Currently root → switch to subgroup
		g.groupPath = g.contextParentPath
		g.parentName = g.contextParentName
	} else {
		// Currently subgroup → switch to root
		g.groupPath = ""
		g.parentName = ""
		g.initPathDefault()
	}
	g.focusIndex = 0
	g.nameInput.Focus()
	g.pathInput.Blur()
	g.validationErr = ""
}

// ShowRename shows the dialog in rename mode
func (g *GroupDialog) ShowRename(currentPath, currentName string) {
	g.visible = true
	g.mode = GroupDialogRename
	g.groupPath = currentPath
	g.validationErr = ""
	g.focusIndex = 0
	g.nameInput.SetValue(currentName)
	g.nameInput.Focus()
	g.pathInput.Blur()
}

// ShowMove shows the dialog for moving a session to a group
func (g *GroupDialog) ShowMove(groups []string) {
	g.visible = true
	g.mode = GroupDialogMove
	g.validationErr = ""
	g.groupNames = groups
	g.selected = 0
}

// ShowRenameSession shows the dialog for renaming a session
func (g *GroupDialog) ShowRenameSession(sessionID, currentName string) {
	g.visible = true
	g.mode = GroupDialogRenameSession
	g.sessionID = sessionID
	g.validationErr = ""
	g.focusIndex = 0
	g.nameInput.SetValue(currentName)
	g.nameInput.Focus()
	g.pathInput.Blur()
}

// GetSessionID returns the session ID being renamed
func (g *GroupDialog) GetSessionID() string {
	return g.sessionID
}

// Hide hides the dialog
func (g *GroupDialog) Hide() {
	g.visible = false
	g.nameInput.Blur()
	g.pathInput.Blur()
}

// IsVisible returns whether the dialog is visible
func (g *GroupDialog) IsVisible() bool {
	return g.visible
}

// Mode returns the current dialog mode
func (g *GroupDialog) Mode() GroupDialogMode {
	return g.mode
}

// GetValue returns the name input value
func (g *GroupDialog) GetValue() string {
	return strings.TrimSpace(g.nameInput.Value())
}

// GetPath returns the path input value (only meaningful for root Create mode)
func (g *GroupDialog) GetPath() string {
	return strings.TrimSpace(g.pathInput.Value())
}

// Validate checks if the dialog values are valid and returns an error message if not
func (g *GroupDialog) Validate() string {
	if g.mode == GroupDialogMove {
		return "" // Move mode doesn't need validation
	}

	name := strings.TrimSpace(g.nameInput.Value())

	// Check for empty name
	if name == "" {
		if g.mode == GroupDialogRenameSession {
			return "Session name cannot be empty"
		}
		return "Project name cannot be empty"
	}

	// Check name length
	if len(name) > MaxNameLength {
		return fmt.Sprintf("Name too long (max %d characters)", MaxNameLength)
	}

	// Check for "/" in group names (would break path hierarchy)
	if g.mode == GroupDialogCreate || g.mode == GroupDialogRename {
		if strings.Contains(name, "/") {
			return "Name cannot contain '/' character"
		}
	}

	// Validate path for root-level project creation
	if g.isRootCreate() {
		path := strings.TrimSpace(g.pathInput.Value())
		if path == "" {
			return "Project path cannot be empty"
		}
	}

	return "" // Valid
}

// SetError sets an inline validation error displayed inside the dialog
func (g *GroupDialog) SetError(msg string) {
	g.validationErr = msg
}

// ClearError clears the inline validation error
func (g *GroupDialog) ClearError() {
	g.validationErr = ""
}

// GetGroupPath returns the group path being edited (or parent path for subgroup creation)
func (g *GroupDialog) GetGroupPath() string {
	return g.groupPath
}

// GetParentPath returns the parent path for subgroup creation
func (g *GroupDialog) GetParentPath() string {
	return g.groupPath
}

// HasParent returns true if creating a subgroup under a parent
func (g *GroupDialog) HasParent() bool {
	return g.groupPath != "" && g.mode == GroupDialogCreate
}

// GetSelectedGroup returns the selected group for move mode
func (g *GroupDialog) GetSelectedGroup() string {
	if g.selected >= 0 && g.selected < len(g.groupNames) {
		return g.groupNames[g.selected]
	}
	return ""
}

// SetSize sets the dialog size
func (g *GroupDialog) SetSize(width, height int) {
	g.width = width
	g.height = height
}

// Update handles input
func (g *GroupDialog) Update(msg tea.KeyMsg) (*GroupDialog, tea.Cmd) {
	if g.mode == GroupDialogMove {
		switch msg.String() {
		case "up", "k":
			if g.selected > 0 {
				g.selected--
			}
		case "down", "j":
			if g.selected < len(g.groupNames)-1 {
				g.selected++
			}
		}
		return g, nil
	}

	// Tab: in root Create mode cycle between name and path; otherwise toggle Root/Subgroup
	if msg.String() == "tab" {
		if g.isRootCreate() {
			// Cycle focus: name ↔ path
			if g.focusIndex == 0 {
				g.focusIndex = 1
				g.nameInput.Blur()
				g.pathInput.Focus()
			} else {
				g.focusIndex = 0
				g.pathInput.Blur()
				g.nameInput.Focus()
			}
			return g, nil
		}
		if g.CanToggle() {
			g.ToggleRootSubgroup()
			return g, nil
		}
	}

	var cmd tea.Cmd
	if g.isRootCreate() && g.focusIndex == 1 {
		g.pathInput, cmd = g.pathInput.Update(msg)
	} else {
		g.nameInput, cmd = g.nameInput.Update(msg)
	}
	return g, cmd
}

// View renders the dialog
func (g *GroupDialog) View() string {
	if !g.visible {
		return ""
	}

	var title string
	var content string

	switch g.mode {
	case GroupDialogCreate:
		if g.parentName != "" {
			title = "Create Subgroup"
			parentInfo := lipgloss.NewStyle().
				Foreground(ColorCyan).
				Render("Parent: " + g.parentName)
			content = parentInfo + "\n\n" + g.nameInput.View()
		} else {
			title = "Create New Project"
			// Render name + path inputs for root-level creation
			activeLabelStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
			labelStyle := lipgloss.NewStyle().Foreground(ColorText)

			var b strings.Builder
			if g.focusIndex == 0 {
				b.WriteString(activeLabelStyle.Render("▶ Name:"))
			} else {
				b.WriteString(labelStyle.Render("  Name:"))
			}
			b.WriteString("\n  ")
			b.WriteString(g.nameInput.View())
			b.WriteString("\n\n")
			if g.focusIndex == 1 {
				b.WriteString(activeLabelStyle.Render("▶ Path:"))
			} else {
				b.WriteString(labelStyle.Render("  Path:"))
			}
			b.WriteString("\n  ")
			b.WriteString(g.pathInput.View())
			content = b.String()
		}

		// Add Root/Subgroup toggle indicator when Tab toggle is available
		if g.CanToggle() {
			activeStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
			dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

			rootTab := "Project"
			subTab := "Subgroup"
			var tabs string
			if g.groupPath == "" {
				// Root mode active
				tabs = activeStyle.Render("["+rootTab+"]") + " ─── " + dimStyle.Render(subTab)
			} else {
				// Subgroup mode active
				tabs = dimStyle.Render(rootTab) + " ─── " + activeStyle.Render("["+subTab+"]")
			}
			content = tabs + "\n\n" + content
		}
	case GroupDialogRename:
		title = "Rename Project"
		content = g.nameInput.View()
	case GroupDialogMove:
		title = "Move to Project"
		var items []string
		for i, name := range g.groupNames {
			if i == g.selected {
				items = append(items, lipgloss.NewStyle().
					Foreground(ColorBg).
					Background(ColorAccent).
					Bold(true).
					Padding(0, 1).
					Render(name))
			} else {
				items = append(items, lipgloss.NewStyle().
					Foreground(ColorText).
					Padding(0, 1).
					Render(name))
			}
		}
		content = strings.Join(items, "\n")
	case GroupDialogRenameSession:
		title = "Rename Session"
		content = g.nameInput.View()
	}

	// Responsive dialog width
	dialogWidth := 50
	if g.width > 0 && g.width < dialogWidth+10 {
		dialogWidth = g.width - 10
		if dialogWidth < 30 {
			dialogWidth = 30
		}
	}
	titleWidth := dialogWidth - 4

	titleStyle := DialogTitleStyle.Width(titleWidth)
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment)
	var hint string
	switch {
	case g.isRootCreate():
		hint = hintStyle.Render("Tab next field │ Enter confirm │ Esc cancel")
	case g.CanToggle():
		hint = hintStyle.Render("Tab toggle │ Enter confirm │ Esc cancel")
	default:
		hint = hintStyle.Render("Enter confirm │ Esc cancel")
	}

	errContent := ""
	if g.validationErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
		errContent = errStyle.Render("⚠ " + g.validationErr)
	}

	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		"",
		content,
		errContent,
		"",
		hint,
	)

	dialog := DialogBoxStyle.
		Width(dialogWidth).
		Render(dialogContent)

	// Center the dialog
	return lipgloss.Place(
		g.width,
		g.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
