package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MCPDialog handles MCP management for Claude sessions
type MCPDialog struct {
	visible     bool
	width       int
	height      int
	projectPath string
	servers     []session.MCPServer
	selected    int
	changed     bool // Track if any changes were made
	err         error
}

// NewMCPDialog creates a new MCP management dialog
func NewMCPDialog() *MCPDialog {
	return &MCPDialog{}
}

// Show displays the MCP dialog for a project
func (m *MCPDialog) Show(projectPath string) error {
	servers, err := session.GetLocalMCPState(projectPath)
	if err != nil {
		return err
	}

	m.visible = true
	m.projectPath = projectPath
	m.servers = servers
	m.selected = 0
	m.changed = false
	m.err = nil

	return nil
}

// Hide hides the dialog
func (m *MCPDialog) Hide() {
	m.visible = false
	m.servers = nil
	m.err = nil
}

// IsVisible returns whether the dialog is visible
func (m *MCPDialog) IsVisible() bool {
	return m.visible
}

// HasServers returns true if there are MCPs to manage
func (m *MCPDialog) HasServers() bool {
	return len(m.servers) > 0
}

// HasChanged returns true if any MCPs were toggled
func (m *MCPDialog) HasChanged() bool {
	return m.changed
}

// GetProjectPath returns the project path being managed
func (m *MCPDialog) GetProjectPath() string {
	return m.projectPath
}

// GetError returns any error that occurred
func (m *MCPDialog) GetError() error {
	return m.err
}

// SetSize sets the dialog size
func (m *MCPDialog) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Toggle toggles the currently selected MCP
func (m *MCPDialog) Toggle() {
	if m.selected >= 0 && m.selected < len(m.servers) {
		name := m.servers[m.selected].Name
		if err := session.ToggleLocalMCP(m.projectPath, name); err != nil {
			m.err = err
			return
		}

		// Update local state
		m.servers[m.selected].Enabled = !m.servers[m.selected].Enabled
		m.changed = true
	}
}

// Update handles input
func (m *MCPDialog) Update(msg tea.KeyMsg) (*MCPDialog, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.servers)-1 {
			m.selected++
		}
	case " ":
		m.Toggle()
	}
	return m, nil
}

// View renders the dialog
func (m *MCPDialog) View() string {
	if !m.visible {
		return ""
	}

	title := "MCP Manager"

	var content string

	if len(m.servers) == 0 {
		content = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true).
			Render("No Local MCPs configured.\nAdd MCPs in .mcp.json")
	} else {
		// Render MCP list
		headerStyle := lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)
		content = headerStyle.Render("Local MCPs (.mcp.json):") + "\n\n"

		for i, server := range m.servers {
			var line string

			// Checkbox
			checkbox := "[ ]"
			if server.Enabled {
				checkbox = "[✓]"
			}

			checkStyle := lipgloss.NewStyle()
			if server.Enabled {
				checkStyle = checkStyle.Foreground(ColorGreen)
			} else {
				checkStyle = checkStyle.Foreground(ColorTextDim)
			}

			nameStyle := lipgloss.NewStyle().Foreground(ColorText)

			line = checkStyle.Render(checkbox) + " " + nameStyle.Render(server.Name)

			// Highlight selected
			if i == m.selected {
				line = lipgloss.NewStyle().
					Background(ColorAccent).
					Foreground(ColorBg).
					Bold(true).
					Padding(0, 1).
					Render(line)
			} else {
				line = lipgloss.NewStyle().
					Padding(0, 1).
					Render(line)
			}

			content += line + "\n"
		}
	}

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(ColorRed)
		content += "\n" + errStyle.Render("Error: "+m.err.Error())
	}

	titleStyle := DialogTitleStyle.Width(44)

	var hint string
	if len(m.servers) > 0 {
		hint = DimStyle.Render("[Space] Toggle  [Enter] Apply  [Esc] Cancel")
	} else {
		hint = DimStyle.Render("[Esc] Close")
	}

	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		"",
		content,
		"",
		hint,
	)

	dialog := DialogBoxStyle.
		Width(48).
		Render(dialogContent)

	// Center the dialog
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
