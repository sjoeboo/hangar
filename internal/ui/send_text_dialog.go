package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SendTextDialog is a simple modal for sending text to a session without attaching.
type SendTextDialog struct {
	visible      bool
	width        int
	height       int
	sessionTitle string
	input        textinput.Model
}

// NewSendTextDialog creates a new send-text dialog
func NewSendTextDialog() *SendTextDialog {
	ti := textinput.New()
	ti.Placeholder = "command or text to send..."
	ti.CharLimit = 500
	ti.Width = 44
	return &SendTextDialog{input: ti}
}

// Show displays the dialog for the given session
func (d *SendTextDialog) Show(sessionTitle string) {
	d.visible = true
	d.sessionTitle = sessionTitle
	d.input.SetValue("")
	d.input.Focus()
}

// Hide hides the dialog
func (d *SendTextDialog) Hide() {
	d.visible = false
	d.input.Blur()
}

// IsVisible returns whether the dialog is visible
func (d *SendTextDialog) IsVisible() bool { return d.visible }

// SetSize sets dialog dimensions for centering
func (d *SendTextDialog) SetSize(w, h int) { d.width = w; d.height = h }

// GetText returns the current input value
func (d *SendTextDialog) GetText() string { return strings.TrimSpace(d.input.Value()) }

// HandleKey returns "confirm", "close", or "" to pass through to the text input.
func (d *SendTextDialog) HandleKey(key string) string {
	switch key {
	case "enter":
		return "confirm"
	case "esc":
		d.Hide()
		return "close"
	}
	return ""
}

// Update passes a message to the text input
func (d *SendTextDialog) Update(msg tea.Msg) {
	d.input, _ = d.input.Update(msg)
}

// View renders the dialog
func (d *SendTextDialog) View() string {
	if !d.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)
	valueStyle := lipgloss.NewStyle().Foreground(ColorAccent)

	dialogWidth := 52
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Padding(1, 2).
		Width(dialogWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Send to Session"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("  Session: "))
	b.WriteString(valueStyle.Render(d.sessionTitle))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("  > "))
	b.WriteString(d.input.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Enter send • Esc cancel"))

	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(b.String()))
}
