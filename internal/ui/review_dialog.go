package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReviewDialog is a two-step dialog for starting a Claude-assisted PR review session.
//
// Step 0: Input — user types a PR # (digits) or branch name.
// Step 1: Confirm — shows resolved branch, session name, and /pr-review prompt before creating.
type ReviewDialog struct {
	visible bool
	width   int
	height  int

	// Project context (set on Show)
	projectName string
	repoDir     string
	groupPath   string // set by home.go before Show

	// Input (step 0)
	input     textinput.Model
	resolving bool // true while async gh lookup is in flight
	errorMsg  string

	// Resolved values (set by SetResolved, moves dialog to step 1)
	step           int
	resolvedBranch string
	prTitle        string
	isPR           bool
	prNumber       string
}

func NewReviewDialog() *ReviewDialog {
	ti := textinput.New()
	ti.Placeholder = "PR # or branch name"
	ti.CharLimit = 200
	return &ReviewDialog{input: ti}
}

func (d *ReviewDialog) Show(projectName, repoDir string) {
	d.visible = true
	d.projectName = projectName
	d.repoDir = repoDir
	d.step = 0
	d.resolving = false
	d.errorMsg = ""
	d.resolvedBranch = ""
	d.prTitle = ""
	d.isPR = false
	d.prNumber = ""
	d.input.Reset()
	d.input.Focus()
}

func (d *ReviewDialog) Hide() {
	d.visible = false
}

func (d *ReviewDialog) IsVisible() bool {
	return d.visible
}

func (d *ReviewDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
	inputWidth := width - 20
	if inputWidth > 60 {
		inputWidth = 60
	}
	if inputWidth < 10 {
		inputWidth = 10
	}
	d.input.Width = inputWidth
}

// SetResolving is called when async PR resolution starts.
func (d *ReviewDialog) SetResolving(resolving bool) {
	d.resolving = resolving
	d.errorMsg = ""
}

// SetResolved advances the dialog to the confirm step with resolved values.
func (d *ReviewDialog) SetResolved(branch, title string, isPR bool, prNum string) {
	d.resolving = false
	d.resolvedBranch = branch
	d.prTitle = title
	d.isPR = isPR
	d.prNumber = prNum
	d.errorMsg = ""
	d.step = 1
}

// SetError displays an inline error (e.g. gh lookup failed).
func (d *ReviewDialog) SetError(msg string) {
	d.resolving = false
	d.errorMsg = msg
}

// IsPRInput returns true if the current input looks like a PR number (all digits).
func (d *ReviewDialog) IsPRInput() bool {
	v := strings.TrimSpace(d.input.Value())
	if v == "" {
		return false
	}
	for _, r := range v {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// GetRawInput returns the trimmed input value.
func (d *ReviewDialog) GetRawInput() string {
	return strings.TrimSpace(d.input.Value())
}

// GetRepoDir returns the project repository root.
func (d *ReviewDialog) GetRepoDir() string {
	return d.repoDir
}

// GetReviewValues returns confirmed review parameters after SetResolved has been called.
// Returns (branch, prNumber, sessionName, initialPrompt).
// initialPrompt is always empty — callers should not auto-send commands on behalf of the user.
func (d *ReviewDialog) GetReviewValues() (branch, prNum, sessionName, initialPrompt string) {
	branch = d.resolvedBranch
	prNum = d.prNumber
	if d.isPR {
		sessionName = "review/pr-" + d.prNumber
	} else {
		sessionName = "review/" + d.resolvedBranch
	}
	return
}

// HandleKey processes a key event. Returns an action string:
//   - "resolve" — step 0 enter with non-empty input
//   - "confirm" — step 1 enter
//   - "cancel"  — esc at step 0
//   - ""        — no action needed
func (d *ReviewDialog) HandleKey(key string) string {
	if d.resolving {
		return ""
	}

	switch key {
	case "esc":
		if d.step == 1 {
			d.step = 0
			d.errorMsg = ""
			d.input.Focus()
			return ""
		}
		d.Hide()
		return "cancel"

	case "enter":
		if d.step == 0 {
			if strings.TrimSpace(d.input.Value()) == "" {
				return ""
			}
			d.errorMsg = ""
			return "resolve"
		}
		if d.step == 1 {
			return "confirm"
		}

	default:
		if d.step == 0 {
			d.errorMsg = ""
		}
	}
	return ""
}

// Update handles bubbletea messages for textinput cursor blink etc.
func (d *ReviewDialog) Update(msg tea.Msg) tea.Cmd {
	if !d.visible || d.step != 0 {
		return nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

func (d *ReviewDialog) View() string {
	if !d.visible {
		return ""
	}

	dialogWidth := d.width - 4
	if dialogWidth > 60 {
		dialogWidth = 60
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorAccent)
	subtleStyle := lipgloss.NewStyle().Foreground(ColorComment)
	errStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	boxBorder := ColorAccent
	if d.errorMsg != "" {
		boxBorder = ColorRed
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(boxBorder).
		Padding(1, 2).
		Width(dialogWidth)

	var content string
	switch d.step {
	case 0:
		content = d.viewInputStep(titleStyle, labelStyle, valueStyle, subtleStyle, errStyle)
	case 1:
		content = d.viewConfirmStep(titleStyle, labelStyle, valueStyle, subtleStyle, errStyle)
	}

	box := boxStyle.Render(content)

	return lipgloss.Place(d.width, d.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func (d *ReviewDialog) viewInputStep(titleStyle, labelStyle, valueStyle, subtleStyle, errStyle lipgloss.Style) string {
	title := titleStyle.Render("Review PR")
	projectLine := labelStyle.Render("Project: ") + valueStyle.Render(d.projectName)
	inputLine := d.input.View()

	var statusLine string
	switch {
	case d.resolving:
		statusLine = subtleStyle.Render("Resolving PR…")
	case d.errorMsg != "":
		statusLine = errStyle.Render(d.errorMsg)
	default:
		statusLine = subtleStyle.Render("enter to continue · esc to cancel")
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, projectLine, inputLine, statusLine)
}

func (d *ReviewDialog) viewConfirmStep(titleStyle, labelStyle, valueStyle, subtleStyle, errStyle lipgloss.Style) string {
	title := titleStyle.Render("Review PR")

	var lines []string
	lines = append(lines, labelStyle.Render("Project: ")+valueStyle.Render(d.projectName))
	if d.isPR && d.prTitle != "" {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("PR #%s: ", d.prNumber))+valueStyle.Render(d.prTitle))
	}
	lines = append(lines, labelStyle.Render("Branch:  ")+valueStyle.Render(d.resolvedBranch))
	_, _, sessionName, _ := d.GetReviewValues()
	lines = append(lines, labelStyle.Render("Session: ")+valueStyle.Render(sessionName))

	var statusLine string
	if d.errorMsg != "" {
		statusLine = errStyle.Render(d.errorMsg)
	} else {
		statusLine = subtleStyle.Render("enter to create · esc to go back")
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s", title, strings.Join(lines, "\n"), statusLine)
}
