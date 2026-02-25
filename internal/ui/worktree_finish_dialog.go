package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// WorktreeFinishDialog handles the two-step worktree finish flow:
// Step 0: Configure options (keep branch)
// Step 1: Confirm the destructive actions
type WorktreeFinishDialog struct {
	visible bool
	width   int
	height  int

	// Session info (set on Show)
	sessionID    string
	sessionTitle string
	branchName   string
	repoRoot     string
	worktreePath string
	isDirty      bool
	dirtyChecked bool // True once async dirty check has returned
	isExecuting  bool // True while finish operation is running
	errorMsg     string

	// Options (step 0)
	keepBranch bool

	// Dialog state
	step int // 0=options, 1=confirm
}

// NewWorktreeFinishDialog creates a new worktree finish dialog
func NewWorktreeFinishDialog() *WorktreeFinishDialog {
	return &WorktreeFinishDialog{}
}

// Show displays the dialog for the given worktree session
func (d *WorktreeFinishDialog) Show(sessionID, sessionTitle, branchName, repoRoot, worktreePath string) {
	d.visible = true
	d.sessionID = sessionID
	d.sessionTitle = sessionTitle
	d.branchName = branchName
	d.repoRoot = repoRoot
	d.worktreePath = worktreePath
	d.isDirty = false
	d.dirtyChecked = false
	d.isExecuting = false
	d.errorMsg = ""
	d.keepBranch = false
	d.step = 0
}

// Hide hides the dialog and resets state
func (d *WorktreeFinishDialog) Hide() {
	d.visible = false
	d.isExecuting = false
	d.errorMsg = ""
}

// IsVisible returns whether the dialog is visible
func (d *WorktreeFinishDialog) IsVisible() bool {
	return d.visible
}

// SetSize sets the dialog dimensions for centering
func (d *WorktreeFinishDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetDirtyStatus updates the dirty check result
func (d *WorktreeFinishDialog) SetDirtyStatus(isDirty bool) {
	d.isDirty = isDirty
	d.dirtyChecked = true
}

// SetError sets an error message on the dialog
func (d *WorktreeFinishDialog) SetError(msg string) {
	d.errorMsg = msg
	d.isExecuting = false
}

// SetExecuting sets the executing state
func (d *WorktreeFinishDialog) SetExecuting(executing bool) {
	d.isExecuting = executing
}

// GetSessionID returns the session ID this dialog is for
func (d *WorktreeFinishDialog) GetSessionID() string {
	return d.sessionID
}

// GetOptions returns the current dialog options
func (d *WorktreeFinishDialog) GetOptions() (keepBranch bool) {
	return d.keepBranch
}

// HandleKey processes a key event and returns the action to take.
// Returns: action string ("close", "confirm", ""), and whether the dialog handled the key.
func (d *WorktreeFinishDialog) HandleKey(key string) (action string) {
	if d.isExecuting {
		return "" // Block input while executing
	}

	if d.step == 1 {
		// Confirm step: y/n/esc
		switch key {
		case "y":
			return "confirm"
		case "n", "esc":
			if d.errorMsg != "" {
				d.errorMsg = ""
				d.step = 0
				return ""
			}
			d.step = 0
			return ""
		}
		return ""
	}

	// Step 0: Options
	switch key {
	case "esc":
		d.Hide()
		return "close"

	case " ":
		d.keepBranch = !d.keepBranch
		return ""

	case "enter":
		d.errorMsg = ""
		d.step = 1
		return ""
	}

	return ""
}

// View renders the dialog
func (d *WorktreeFinishDialog) View() string {
	if !d.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorAccent)
	checkboxStyle := lipgloss.NewStyle().Foreground(ColorText)
	footerStyle := lipgloss.NewStyle().Foreground(ColorComment)
	errStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	// Responsive dialog width
	dialogWidth := 48
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 35 {
			dialogWidth = 35
		}
	}

	boxBorder := ColorAccent
	if d.errorMsg != "" {
		boxBorder = ColorRed
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(boxBorder).
		Padding(1, 2).
		Width(dialogWidth)

	if d.step == 1 {
		return d.viewConfirm(titleStyle, labelStyle, errStyle, footerStyle, boxStyle)
	}

	return d.viewOptions(titleStyle, labelStyle, valueStyle, checkboxStyle, footerStyle, boxStyle)
}

func (d *WorktreeFinishDialog) viewOptions(titleStyle, labelStyle, valueStyle, checkboxStyle, footerStyle lipgloss.Style, boxStyle lipgloss.Style) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Finish Worktree"))
	b.WriteString("\n\n")

	// Session info
	b.WriteString(labelStyle.Render("  Session:  "))
	b.WriteString(valueStyle.Render(d.sessionTitle))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("  Branch:   "))
	branchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	b.WriteString(branchStyle.Render(d.branchName))
	b.WriteString("\n")

	// Dirty status
	b.WriteString(labelStyle.Render("  Status:   "))
	if !d.dirtyChecked {
		b.WriteString(labelStyle.Render("checking..."))
	} else if d.isDirty {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString(warnStyle.Render("dirty (uncommitted changes)"))
	} else {
		cleanStyle := lipgloss.NewStyle().Foreground(ColorGreen)
		b.WriteString(cleanStyle.Render("clean"))
	}
	b.WriteString("\n\n")

	// Keep branch checkbox
	keepCheck := "[ ]"
	if d.keepBranch {
		keepCheck = "[x]"
	}
	b.WriteString(checkboxStyle.Render(fmt.Sprintf("  %s Keep branch after cleanup", keepCheck)))
	b.WriteString("\n")

	// Error line
	if d.errorMsg != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render("  " + d.errorMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("Space toggle | Enter confirm | Esc cancel"))

	dialog := boxStyle.Render(b.String())
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
}

func (d *WorktreeFinishDialog) viewConfirm(titleStyle, labelStyle, errStyle, footerStyle lipgloss.Style, boxStyle lipgloss.Style) string {
	var b strings.Builder

	if d.isExecuting {
		b.WriteString(titleStyle.Render("Finishing Worktree..."))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("  Please wait..."))
		dialog := boxStyle.Render(b.String())
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if d.errorMsg != "" {
		b.WriteString(errStyle.Render("Finish Failed"))
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render("  " + d.errorMsg))
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render("n back | Esc cancel"))
		dialog := boxStyle.Render(b.String())
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	b.WriteString(titleStyle.Render("Confirm"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("  This will:"))
	b.WriteString("\n")

	actionStyle := lipgloss.NewStyle().Foreground(ColorText)
	b.WriteString(actionStyle.Render("  • Remove worktree directory"))
	b.WriteString("\n")
	if !d.keepBranch {
		b.WriteString(actionStyle.Render(fmt.Sprintf("  • Delete branch %s", d.branchName)))
		b.WriteString("\n")
	}
	b.WriteString(actionStyle.Render("  • Remove session from hangar"))
	b.WriteString("\n")

	// Dirty warning
	if d.isDirty {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
		b.WriteString("\n")
		b.WriteString(warnStyle.Render("  ⚠ Worktree has uncommitted changes!"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("y Finish | n Cancel"))

	dialog := boxStyle.Render(b.String())
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
}
