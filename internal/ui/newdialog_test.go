package ui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewNewDialog(t *testing.T) {
	d := NewNewDialog()

	if d == nil {
		t.Fatal("NewNewDialog returned nil")
	}
	if d.IsVisible() {
		t.Error("Dialog should not be visible by default")
	}
	if len(d.presetCommands) == 0 {
		t.Error("presetCommands should not be empty")
	}
}

func TestDialogVisibility(t *testing.T) {
	d := NewNewDialog()

	d.Show()
	if !d.IsVisible() {
		t.Error("Dialog should be visible after Show()")
	}

	d.Hide()
	if d.IsVisible() {
		t.Error("Dialog should not be visible after Hide()")
	}
}

func TestDialogSetSize(t *testing.T) {
	d := NewNewDialog()
	d.SetSize(100, 50)

	if d.width != 100 {
		t.Errorf("Width = %d, want 100", d.width)
	}
	if d.height != 50 {
		t.Errorf("Height = %d, want 50", d.height)
	}
}

func TestDialogPresetCommands(t *testing.T) {
	d := NewNewDialog()

	// Should have shell (empty), claude, gemini, opencode, codex
	expectedCommands := []string{"", "claude", "gemini", "opencode", "codex"}

	if len(d.presetCommands) != len(expectedCommands) {
		t.Errorf("Expected %d preset commands, got %d", len(expectedCommands), len(d.presetCommands))
	}

	for i, cmd := range expectedCommands {
		if d.presetCommands[i] != cmd {
			t.Errorf("presetCommands[%d] = %s, want %s", i, d.presetCommands[i], cmd)
		}
	}
}

func TestDialogGetValues(t *testing.T) {
	d := NewNewDialog()
	d.nameInput.SetValue("my-session")
	d.pathInput.SetValue("/tmp/project")
	d.commandCursor = 1 // claude

	name, path, command := d.GetValues()

	if name != "my-session" {
		t.Errorf("name = %s, want my-session", name)
	}
	if path != "/tmp/project" {
		t.Errorf("path = %s, want /tmp/project", path)
	}
	if command != "claude" {
		t.Errorf("command = %s, want claude", command)
	}
}

func TestDialogExpandTilde(t *testing.T) {
	d := NewNewDialog()
	d.nameInput.SetValue("test")
	d.pathInput.SetValue("~/projects")

	_, path, _ := d.GetValues()

	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(path, home) {
		t.Errorf("path should expand ~ to home directory, got %s", path)
	}
}

func TestDialogView(t *testing.T) {
	d := NewNewDialog()

	// Not visible - should return empty
	view := d.View()
	if view != "" {
		t.Error("View should be empty when not visible")
	}

	// Visible - should return content
	d.SetSize(80, 24)
	d.Show()
	view = d.View()
	if view == "" {
		t.Error("View should not be empty when visible")
	}
	if !strings.Contains(view, "New Session") {
		t.Error("View should contain 'New Session' title")
	}
}

func TestNewDialog_SetPathSuggestions(t *testing.T) {
	d := NewNewDialog()

	paths := []string{
		"/Users/test/project1",
		"/Users/test/project2",
		"/Users/test/other",
	}

	d.SetPathSuggestions(paths)

	if len(d.pathSuggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(d.pathSuggestions))
	}

	// Verify suggestions are set on textinput
	available := d.pathInput.AvailableSuggestions()
	if len(available) != 3 {
		t.Errorf("expected 3 available suggestions on pathInput, got %d", len(available))
	}
}

func TestNewDialog_ShowSuggestionsEnabled(t *testing.T) {
	d := NewNewDialog()

	// ShowSuggestions should be enabled by default
	if !d.pathInput.ShowSuggestions {
		t.Error("expected ShowSuggestions to be true on pathInput")
	}
}

func TestNewDialog_SuggestionFiltering(t *testing.T) {
	d := NewNewDialog()

	paths := []string{
		"/Users/test/project-alpha",
		"/Users/test/project-beta",
		"/Users/test/other-thing",
	}

	d.SetPathSuggestions(paths)

	// Verify suggestions are available
	available := d.pathInput.AvailableSuggestions()
	if len(available) != 3 {
		t.Errorf("expected 3 available suggestions, got %d", len(available))
	}

	// Verify specific suggestions are in the list
	hasProjectAlpha := false
	hasProjectBeta := false
	hasOtherThing := false
	for _, s := range available {
		if s == "/Users/test/project-alpha" {
			hasProjectAlpha = true
		}
		if s == "/Users/test/project-beta" {
			hasProjectBeta = true
		}
		if s == "/Users/test/other-thing" {
			hasOtherThing = true
		}
	}

	if !hasProjectAlpha || !hasProjectBeta || !hasOtherThing {
		t.Error("not all expected suggestions are available")
	}
}

func TestNewDialog_MalformedPathFix(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal tilde path",
			input:    "~/projects/myapp",
			expected: home + "/projects/myapp",
		},
		{
			name:     "malformed path with cwd prefix",
			input:    "/Users/someone/claude-deck~/projects/myapp",
			expected: home + "/projects/myapp",
		},
		{
			name:     "already expanded path",
			input:    "/Users/ashesh/projects/myapp",
			expected: "/Users/ashesh/projects/myapp",
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: home,
		},
		{
			name:     "malformed path with different prefix",
			input:    "/some/random/path~/other/path",
			expected: home + "/other/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewNewDialog()
			d.pathInput.SetValue(tt.input)

			_, path, _ := d.GetValues()

			if path != tt.expected {
				t.Errorf("GetValues() path = %q, want %q", path, tt.expected)
			}
		})
	}
}

// TestNewDialog_TabDoesNotOverwriteCustomPath tests Issue #22:
// When user enters a new folder path and presses Tab to move to agent selection,
// the custom path should NOT be overwritten by a suggestion.
func TestNewDialog_TabDoesNotOverwriteCustomPath(t *testing.T) {
	d := NewNewDialog()
	d.Show() // Dialog must be visible for Update to process keys

	// Set up suggestions (simulating previously used paths)
	suggestions := []string{
		"/Users/test/old-project-1",
		"/Users/test/old-project-2",
	}
	d.SetPathSuggestions(suggestions)

	// User is on path field (focusIndex 1)
	d.focusIndex = 1
	d.updateFocus()

	// User types a completely NEW path that doesn't match any suggestion
	customPath := "/Users/test/brand-new-project"
	d.pathInput.SetValue(customPath)

	// User presses Tab to move to command selection
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})

	// The custom path should be PRESERVED, not overwritten
	_, path, _ := d.GetValues()

	if path != customPath {
		t.Errorf("Tab overwrote custom path!\nGot: %q\nWant: %q\nThis is the bug from Issue #22", path, customPath)
	}

	// Focus should have moved to command field
	if d.focusIndex != 2 {
		t.Errorf("focusIndex = %d, want 2 (command field)", d.focusIndex)
	}
}

// TestNewDialog_TabAppliesSuggestionWhenNavigated tests that Tab DOES apply
// the suggestion when the user explicitly navigated to one using Ctrl+N/P.
func TestNewDialog_TabAppliesSuggestionWhenNavigated(t *testing.T) {
	d := NewNewDialog()
	d.Show()

	suggestions := []string{
		"/Users/test/project-1",
		"/Users/test/project-2",
	}
	d.SetPathSuggestions(suggestions)

	// User is on path field
	d.focusIndex = 1
	d.updateFocus()

	// User types something, then navigates to suggestion with Ctrl+N
	d.pathInput.SetValue("/some/partial")
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyCtrlN})

	// Now Tab should apply the suggestion
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})

	_, path, _ := d.GetValues()

	// Should be the second suggestion (Ctrl+N moved from 0 to 1)
	if path != "/Users/test/project-2" {
		t.Errorf("Tab should apply suggestion after Ctrl+N navigation\nGot: %q\nWant: %q", path, "/Users/test/project-2")
	}
}

// TestNewDialog_TypingResetsSuggestionNavigation tests that typing after
// navigating suggestions resets the navigation state.
func TestNewDialog_TypingResetsSuggestionNavigation(t *testing.T) {
	d := NewNewDialog()
	d.Show()

	suggestions := []string{
		"/Users/test/project-1",
		"/Users/test/project-2",
	}
	d.SetPathSuggestions(suggestions)

	d.focusIndex = 1
	d.updateFocus()

	// User navigates to a suggestion
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyCtrlN})

	// Verify navigation flag is set
	if !d.suggestionNavigated {
		t.Error("suggestionNavigated should be true after Ctrl+N")
	}

	// User then types something new - simulate by sending a key
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Flag should be reset
	if d.suggestionNavigated {
		t.Error("suggestionNavigated should be false after typing")
	}

	// Set a custom path and press Tab
	d.pathInput.SetValue("/my/new/path")
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})

	_, path, _ := d.GetValues()

	if path != "/my/new/path" {
		t.Errorf("Typing should reset suggestion navigation\nGot: %q\nWant: %q", path, "/my/new/path")
	}
}
