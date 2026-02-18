package ui

import (
	"sort"
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/session"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SkillColumn identifies the focused column.
type SkillColumn int

const (
	SkillColumnAttached SkillColumn = iota
	SkillColumnAvailable
)

// SkillDialogItem wraps one discovered skill.
type SkillDialogItem struct {
	Candidate session.SkillCandidate
}

// SkillDialog manages project-scoped Claude skills.
type SkillDialog struct {
	visible     bool
	width       int
	height      int
	projectPath string
	sessionID   string
	tool        string

	column SkillColumn

	attached      []SkillDialogItem
	available     []SkillDialogItem
	attachedIdx   int
	availableIdx  int
	hasChanges    bool
	err           error
	emptyHelpText string
}

// NewSkillDialog creates a skill manager dialog instance.
func NewSkillDialog() *SkillDialog {
	return &SkillDialog{}
}

// Show opens the dialog for a specific project/session.
func (d *SkillDialog) Show(projectPath, sessionID, tool string) error {
	d.projectPath = projectPath
	d.sessionID = sessionID
	d.tool = tool
	d.err = nil
	d.hasChanges = false
	d.column = SkillColumnAttached
	d.attachedIdx = 0
	d.availableIdx = 0
	d.emptyHelpText = ""

	if tool != "claude" {
		d.visible = true
		d.attached = nil
		d.available = nil
		d.emptyHelpText = "Skills manager is currently available for Claude sessions only."
		return nil
	}

	availableSkills, err := session.ListAvailableSkills()
	if err != nil {
		return err
	}
	attachedSkills, err := session.GetAttachedProjectSkills(projectPath)
	if err != nil {
		return err
	}

	discoveredByID := make(map[string]session.SkillCandidate, len(availableSkills))
	for _, skill := range availableSkills {
		discoveredByID[skill.ID] = skill
	}

	d.attached = make([]SkillDialogItem, 0, len(attachedSkills))
	attachedIDs := make(map[string]bool, len(attachedSkills))
	for _, attachment := range attachedSkills {
		candidate, ok := discoveredByID[attachment.ID]
		if !ok {
			candidate = session.SkillCandidate{
				ID:          attachment.ID,
				Name:        attachment.Name,
				Source:      attachment.Source,
				SourcePath:  attachment.SourcePath,
				EntryName:   attachment.EntryName,
				Description: "(source unavailable)",
				Kind:        "dir",
			}
		}
		d.attached = append(d.attached, SkillDialogItem{Candidate: candidate})
		attachedIDs[candidate.ID] = true
	}

	d.available = make([]SkillDialogItem, 0, len(availableSkills))
	for _, candidate := range availableSkills {
		if attachedIDs[candidate.ID] {
			continue
		}
		d.available = append(d.available, SkillDialogItem{Candidate: candidate})
	}

	sort.Slice(d.attached, func(i, j int) bool {
		return strings.ToLower(d.attached[i].Candidate.Name) < strings.ToLower(d.attached[j].Candidate.Name)
	})
	sort.Slice(d.available, func(i, j int) bool {
		return strings.ToLower(d.available[i].Candidate.Name) < strings.ToLower(d.available[j].Candidate.Name)
	})

	d.visible = true
	return nil
}

// Hide closes the dialog.
func (d *SkillDialog) Hide() {
	d.visible = false
	d.attached = nil
	d.available = nil
	d.err = nil
	d.emptyHelpText = ""
}

// IsVisible returns whether dialog is shown.
func (d *SkillDialog) IsVisible() bool {
	return d.visible
}

// SetSize updates dialog dimensions.
func (d *SkillDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// HasChanged indicates whether user moved any item.
func (d *SkillDialog) HasChanged() bool {
	return d.hasChanges
}

// GetSessionID returns the managed session ID.
func (d *SkillDialog) GetSessionID() string {
	return d.sessionID
}

// GetError returns the latest apply error.
func (d *SkillDialog) GetError() error {
	return d.err
}

func (d *SkillDialog) currentListAndIndex() (*[]SkillDialogItem, *int) {
	if d.column == SkillColumnAttached {
		return &d.attached, &d.attachedIdx
	}
	return &d.available, &d.availableIdx
}

// Move toggles one item between attached and available lists.
func (d *SkillDialog) Move() {
	list, idx := d.currentListAndIndex()
	if len(*list) == 0 || *idx < 0 || *idx >= len(*list) {
		return
	}

	item := (*list)[*idx]
	*list = append((*list)[:*idx], (*list)[*idx+1:]...)

	if d.column == SkillColumnAttached {
		d.available = append(d.available, item)
		sort.Slice(d.available, func(i, j int) bool {
			return strings.ToLower(d.available[i].Candidate.Name) < strings.ToLower(d.available[j].Candidate.Name)
		})
	} else {
		d.attached = append(d.attached, item)
		sort.Slice(d.attached, func(i, j int) bool {
			return strings.ToLower(d.attached[i].Candidate.Name) < strings.ToLower(d.attached[j].Candidate.Name)
		})
	}

	d.hasChanges = true
	if *idx >= len(*list) && len(*list) > 0 {
		*idx = len(*list) - 1
	}
}

// Apply saves project skills according to attached column state.
func (d *SkillDialog) Apply() error {
	d.err = nil
	if d.tool != "claude" {
		return nil
	}

	desired := make([]session.SkillCandidate, 0, len(d.attached))
	for _, item := range d.attached {
		desired = append(desired, item.Candidate)
	}

	if err := session.ApplyProjectSkills(d.projectPath, desired); err != nil {
		d.err = err
		return err
	}
	return nil
}

// Update handles keyboard input while dialog is visible.
func (d *SkillDialog) Update(msg tea.KeyMsg) (*SkillDialog, tea.Cmd) {
	list, idx := d.currentListAndIndex()

	switch msg.String() {
	case "left", "h":
		d.column = SkillColumnAttached
	case "right", "l":
		d.column = SkillColumnAvailable
	case "up", "k":
		if len(*list) > 0 && *idx > 0 {
			*idx--
		}
	case "down", "j":
		if len(*list) > 0 && *idx < len(*list)-1 {
			*idx++
		}
	case " ":
		d.Move()
	}

	return d, nil
}

func (d *SkillDialog) renderColumn(title string, items []SkillDialogItem, selectedIdx int, focused bool) string {
	headerStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	if focused {
		headerStyle = headerStyle.Foreground(ColorAccent)
	}
	header := headerStyle.Render("- " + title + " ")

	colWidth := 38
	headerLen := len("- " + title + " ")
	headerPad := colWidth - headerLen
	if headerPad > 0 {
		header += headerStyle.Render(repeatStr("-", headerPad))
	}

	lines := []string{header}
	if len(items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true).Render("  (empty)"))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	for i, item := range items {
		label := item.Candidate.Name
		if item.Candidate.Source != "" {
			label += " [" + item.Candidate.Source + "]"
		}
		if len(label) > colWidth-4 {
			label = label[:colWidth-7] + "..."
		}

		if i == selectedIdx && focused {
			lines = append(lines, lipgloss.NewStyle().
				Background(ColorAccent).
				Foreground(ColorBg).
				Bold(true).
				Width(colWidth).
				Render(" > "+label))
		} else {
			lines = append(lines, lipgloss.NewStyle().
				Foreground(ColorText).
				Width(colWidth).
				Render("   "+label))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (d *SkillDialog) renderEmptyStateHelp() string {
	helpStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	highlightStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	pathStyle := lipgloss.NewStyle().Foreground(ColorCyan)

	lines := []string{
		"",
		highlightStyle.Render("No discoverable skills found"),
		"",
		helpStyle.Render("Add a source with:"),
		pathStyle.Render("  agent-deck skill source add <name> <path>"),
		"",
		helpStyle.Render("Or place reusable skills in:"),
		pathStyle.Render("  ~/.agent-deck/skills/pool"),
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// View renders the dialog body.
func (d *SkillDialog) View() string {
	if !d.visible {
		return ""
	}

	title := "Skills Manager"
	scopeDesc := DimStyle.Render("Writes to: .agent-deck/skills.toml + .claude/skills (project)")
	if d.tool != "claude" && d.emptyHelpText != "" {
		scopeDesc = lipgloss.NewStyle().Foreground(ColorYellow).Render(d.emptyHelpText)
	}

	attachedCol := d.renderColumn("Attached", d.attached, d.attachedIdx, d.column == SkillColumnAttached)
	availableCol := d.renderColumn("Available", d.available, d.availableIdx, d.column == SkillColumnAvailable)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, attachedCol, "  ", availableCol)

	hint := lipgloss.NewStyle().Foreground(ColorComment).Render("←→ column │ Space move │ Enter apply │ Esc cancel")

	dialogWidth := 86
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 56 {
			dialogWidth = 56
		}
	}
	titleWidth := dialogWidth - 4

	parts := []string{
		DialogTitleStyle.Width(titleWidth).Render(title),
		"",
		scopeDesc,
		"",
	}

	if len(d.attached) == 0 && len(d.available) == 0 {
		parts = append(parts, d.renderEmptyStateHelp())
	} else {
		parts = append(parts, columns)
	}

	if d.err != nil {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(ColorRed).Render("Error: "+d.err.Error()))
	}
	parts = append(parts, "", hint)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return DialogBoxStyle.Width(dialogWidth).Render(content)
}
