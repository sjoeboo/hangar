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

// GroupDialog handles project/session creation, renaming, and moving sessions.
// "Groups" in the data model are always top-level projects; subgroups are no longer supported.
type GroupDialog struct {
	visible    bool
	mode       GroupDialogMode
	nameInput  textinput.Model
	pathInput  textinput.Model // Create mode: base directory for the new project
	focusIndex int             // 0=name, 1=path (Create mode only)
	width      int
	height     int
	groupPath  string   // Path of group being renamed (Rename mode only)
	groupNames []string // Available groups (Move mode only)
	selected   int      // Selected group index (Move mode only)
	sessionID  string   // Session ID being renamed (RenameSession mode only)

	validationErr string // Inline validation error displayed inside the dialog
}

// NewGroupDialog creates a new group dialog
func NewGroupDialog() *GroupDialog {
	ni := textinput.New()
	ni.Placeholder = "Project name"
	ni.CharLimit = 50
	ni.Width = 44

	pi := textinput.New()
	pi.Placeholder = "~/path/to/repo"
	pi.CharLimit = 256
	pi.Width = 44

	return &GroupDialog{
		nameInput:  ni,
		pathInput:  pi,
		groupNames: []string{},
	}
}

// Show shows the dialog in create-project mode.
// The path input is pre-populated with the current working directory.
func (g *GroupDialog) Show() {
	g.visible = true
	g.mode = GroupDialogCreate
	g.groupPath = ""
	g.focusIndex = 0
	g.validationErr = ""
	g.nameInput.SetValue("")
	g.nameInput.Focus()
	g.pathInput.Blur()
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

// ShowRename shows the dialog in rename mode
func (g *GroupDialog) ShowRename(currentPath, currentName string) {
	g.visible = true
	g.mode = GroupDialogRename
	g.groupPath = currentPath
	g.focusIndex = 0
	g.validationErr = ""
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
	g.focusIndex = 0
	g.validationErr = ""
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

// GetPath returns the path input value (only meaningful for Create mode)
func (g *GroupDialog) GetPath() string {
	return strings.TrimSpace(g.pathInput.Value())
}

// Validate checks if the dialog values are valid and returns an error message if not
func (g *GroupDialog) Validate() string {
	if g.mode == GroupDialogMove {
		return ""
	}

	name := strings.TrimSpace(g.nameInput.Value())

	if name == "" {
		if g.mode == GroupDialogRenameSession {
			return "Session name cannot be empty"
		}
		return "Project name cannot be empty"
	}

	if len(name) > MaxNameLength {
		return fmt.Sprintf("Name too long (max %d characters)", MaxNameLength)
	}

	if g.mode == GroupDialogCreate || g.mode == GroupDialogRename {
		if strings.Contains(name, "/") {
			return "Name cannot contain '/' character"
		}
	}

	if g.mode == GroupDialogCreate {
		if strings.TrimSpace(g.pathInput.Value()) == "" {
			return "Project path cannot be empty"
		}
	}

	return ""
}

// SetError sets an inline validation error displayed inside the dialog
func (g *GroupDialog) SetError(msg string) {
	g.validationErr = msg
}

// ClearError clears the inline validation error
func (g *GroupDialog) ClearError() {
	g.validationErr = ""
}

// GetGroupPath returns the path of the group being renamed
func (g *GroupDialog) GetGroupPath() string {
	return g.groupPath
}

// GetSelectedGroup returns the selected group for move mode
func (g *GroupDialog) GetSelectedGroup() string {
	if g.selected >= 0 && g.selected < len(g.groupNames) {
		return g.groupNames[g.selected]
	}
	return ""
}

// SetSize sets the dialog size and updates textinput widths to fill the dialog.
func (g *GroupDialog) SetSize(width, height int) {
	g.width = width
	g.height = height

	dialogWidth := 52
	if width > 0 && width < dialogWidth+10 {
		dialogWidth = width - 10
		if dialogWidth < 32 {
			dialogWidth = 32
		}
	}
	// textinput Width is the field area; the "> " prompt adds 2 chars.
	// Subtract 2 for the prompt and 2 for a right margin inside the padded dialog.
	inputWidth := dialogWidth - 4
	if inputWidth < 20 {
		inputWidth = 20
	}
	g.nameInput.Width = inputWidth
	g.pathInput.Width = inputWidth
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

	// Tab cycles between name and path in Create mode
	if msg.String() == "tab" && g.mode == GroupDialogCreate {
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

	var cmd tea.Cmd
	if g.mode == GroupDialogCreate && g.focusIndex == 1 {
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

	// Responsive dialog width (must match SetSize logic)
	dialogWidth := 52
	if g.width > 0 && g.width < dialogWidth+10 {
		dialogWidth = g.width - 10
		if dialogWidth < 32 {
			dialogWidth = 32
		}
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPurple).
		Background(ColorSurface).
		Padding(1, 2).
		Width(dialogWidth)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPurple).
		Width(dialogWidth).
		Align(lipgloss.Center)

	activeLabelStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment)
	errStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	var b strings.Builder

	switch g.mode {
	case GroupDialogCreate:
		b.WriteString(titleStyle.Render("Create New Project"))
		b.WriteString("\n\n")

		if g.focusIndex == 0 {
			b.WriteString(activeLabelStyle.Render("▶ Name:"))
		} else {
			b.WriteString(labelStyle.Render("  Name:"))
		}
		b.WriteString("\n")
		b.WriteString(g.nameInput.View())
		b.WriteString("\n\n")

		if g.focusIndex == 1 {
			b.WriteString(activeLabelStyle.Render("▶ Path:"))
		} else {
			b.WriteString(labelStyle.Render("  Path:"))
		}
		b.WriteString("\n")
		b.WriteString(g.pathInput.View())

		if g.validationErr != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render("⚠ " + g.validationErr))
		}

		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("Tab  next field  │  Enter  confirm  │  Esc  cancel"))

	case GroupDialogRename:
		b.WriteString(titleStyle.Render("Rename Project"))
		b.WriteString("\n\n")
		b.WriteString(activeLabelStyle.Render("▶ Name:"))
		b.WriteString("\n")
		b.WriteString(g.nameInput.View())

		if g.validationErr != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render("⚠ " + g.validationErr))
		}

		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("Enter  confirm  │  Esc  cancel"))

	case GroupDialogMove:
		b.WriteString(titleStyle.Render("Move to Project"))
		b.WriteString("\n\n")

		for i, name := range g.groupNames {
			if i > 0 {
				b.WriteString("\n")
			}
			if i == g.selected {
				b.WriteString(lipgloss.NewStyle().
					Foreground(ColorBg).
					Background(ColorAccent).
					Bold(true).
					Padding(0, 1).
					Render(name))
			} else {
				b.WriteString(lipgloss.NewStyle().
					Foreground(ColorText).
					Padding(0, 1).
					Render(name))
			}
		}

		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("↑/↓  navigate  │  Enter  confirm  │  Esc  cancel"))

	case GroupDialogRenameSession:
		b.WriteString(titleStyle.Render("Rename Session"))
		b.WriteString("\n\n")
		b.WriteString(activeLabelStyle.Render("▶ Name:"))
		b.WriteString("\n")
		b.WriteString(g.nameInput.View())

		if g.validationErr != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render("⚠ " + g.validationErr))
		}

		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("Enter  confirm  │  Esc  cancel"))
	}

	dialog := dialogStyle.Render(b.String())

	return lipgloss.Place(
		g.width,
		g.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
