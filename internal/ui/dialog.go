package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Dialog is the common interface implemented by all modal dialogs in the TUI.
// It provides a uniform contract for visibility, rendering, key handling, and sizing,
// enabling the DialogManager (see dialog_manager.go) to route input without
// knowing the concrete dialog type.
type Dialog interface {
	// IsVisible reports whether the dialog is currently shown.
	IsVisible() bool

	// Show makes the dialog visible.
	Show()

	// Hide dismisses the dialog.
	Hide()

	// View renders the dialog to a string. Called only when IsVisible() is true.
	View() string

	// HandleKey processes a key event. Returns the resulting command and whether
	// the key was consumed (true = stop further processing).
	HandleKey(key tea.KeyMsg) (cmd tea.Cmd, consumed bool)

	// SetSize informs the dialog of the current terminal dimensions.
	SetSize(width, height int)
}
