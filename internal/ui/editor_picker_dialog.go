package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sjoeboo/hangar/internal/editor"
)

// EditorPickerDialog lets the user pick an editor to open the current worktree directory.
// Pressing `e` fast-launches the configured editor directly;
// pressing `E` opens this dialog to choose from built-in presets or the custom command.
type EditorPickerDialog struct {
	visible      bool
	width        int
	height       int
	cursor       int
	worktreePath string // directory to open in the chosen editor
	tmuxSession  string // tmux session name, needed for terminal editors
	configEditor string // resolved editor cmd (HANGAR_EDITOR wins over config)
}

// NewEditorPickerDialog returns a zero-value dialog ready to use.
func NewEditorPickerDialog() *EditorPickerDialog {
	return &EditorPickerDialog{}
}

// Show opens the dialog for a specific worktree.
// configEditor should already be resolved via editor.GetCmd(userConfig.Editor).
func (d *EditorPickerDialog) Show(worktreePath, tmuxSession, configEditor string) {
	d.visible = true
	d.worktreePath = worktreePath
	d.tmuxSession = tmuxSession
	d.configEditor = configEditor
	d.cursor = 0
}

// Hide closes the dialog.
func (d *EditorPickerDialog) Hide() { d.visible = false }

// IsVisible reports whether the dialog is open.
func (d *EditorPickerDialog) IsVisible() bool { return d.visible }

// SetSize updates the screen dimensions used for centering.
func (d *EditorPickerDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// HandleKey processes a navigation or selection keypress.
// Returns (true, editorCmd) when the user confirms a choice; editorCmd may be empty
// if the "Custom" entry is selected but no editor is configured.
func (d *EditorPickerDialog) HandleKey(key string) (selected bool, editorCmd string) {
	presets := editor.Presets
	switch key {
	case "j", "down":
		if d.cursor < len(presets)-1 {
			d.cursor++
		}
	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}
	case "enter", " ":
		p := presets[d.cursor]
		cmd := p.Cmd
		if p.Name == "custom" {
			cmd = d.configEditor // empty when none configured
		}
		return true, cmd
	}
	return false, ""
}

// View renders the picker overlay, centered on screen.
func (d *EditorPickerDialog) View() string {
	if !d.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	selectedLabelStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	normalLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	footerStyle := lipgloss.NewStyle().Foreground(ColorComment).Italic(true)

	var b strings.Builder
	b.WriteString(titleStyle.Render("OPEN IN EDITOR"))
	b.WriteString("\n\n")

	for i, p := range editor.Presets {
		var hint string
		if p.Name == "custom" {
			if d.configEditor != "" {
				hint = "  " + dimStyle.Render("("+d.configEditor+")")
			} else {
				hint = "  " + dimStyle.Render("(not configured)")
			}
		}

		if i == d.cursor {
			b.WriteString("▶ " + selectedLabelStyle.Render(p.Label) + hint)
		} else {
			b.WriteString("  " + normalLabelStyle.Render(p.Label) + hint)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("j/k navigate • enter select • esc cancel"))

	box := DialogBoxStyle.Width(40).Render(b.String())
	return centerInScreen(box, d.width, d.height)
}
