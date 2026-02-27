package ui

import (
	"fmt"
	"strings"

	"github.com/sjoeboo/hangar/internal/git"
	"github.com/sjoeboo/hangar/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// kanbanColumn is a derived view structure grouping todos by status for kanban rendering.
type kanbanColumn struct {
	status session.TodoStatus
	label  string
	todos  []*session.Todo
}

// orderedStatuses defines the fixed left-to-right column order for the kanban board.
var orderedStatuses = []session.TodoStatus{
	session.TodoStatusTodo,
	session.TodoStatusInProgress,
	session.TodoStatusInReview,
	session.TodoStatusDone,
}

// buildColumns groups todos into kanban columns ordered by status.
// Always returns the 4 main columns; appends an Orphaned column only when orphaned todos exist.
func buildColumns(todos []*session.Todo) []kanbanColumn {
	statusIndex := map[session.TodoStatus]int{}
	cols := make([]kanbanColumn, len(orderedStatuses))
	for i, s := range orderedStatuses {
		cols[i] = kanbanColumn{status: s, label: todoStatusLabel(s)}
		statusIndex[s] = i
	}
	var orphaned []*session.Todo
	for _, t := range todos {
		if idx, ok := statusIndex[t.Status]; ok {
			cols[idx].todos = append(cols[idx].todos, t)
		} else {
			orphaned = append(orphaned, t)
		}
	}
	if len(orphaned) > 0 {
		cols = append(cols, kanbanColumn{
			status: session.TodoStatusOrphaned,
			label:  todoStatusLabel(session.TodoStatusOrphaned),
			todos:  orphaned,
		})
	}
	return cols
}

// todoDialogMode controls which sub-view is active.
type todoDialogMode int

const (
	todoModeKanban todoDialogMode = iota
	todoModeNew                   // new-todo form
	todoModeEdit                  // edit-todo form
	todoModeStatus                // status picker
)

// TodoDialog shows and manages todos for a project.
type TodoDialog struct {
	visible     bool
	width       int
	height      int
	projectPath string
	groupPath   string
	groupName   string
	todos       []*session.Todo

	// kanban cursor state (derived cols cached here, rebuilt on SetTodos/Show)
	cols        []kanbanColumn
	selectedCol int
	selectedRow []int // one cursor per column; indexed by column index

	mode     todoDialogMode
	errorMsg string

	// new/edit form fields
	titleInput    textinput.Model
	descInput     textinput.Model
	promptInput   textinput.Model
	formFocus     int               // 0=title, 1=desc, 2=prompt
	editingID     string            // non-empty when editing
	newTodoStatus session.TodoStatus // status pre-selected from focused column when pressing n

	// status picker
	statusOptions []session.TodoStatus
	statusCursor  int
}

// NewTodoDialog creates a new TodoDialog.
func NewTodoDialog() *TodoDialog {
	titleInput := textinput.New()
	titleInput.Placeholder = "Todo title"
	titleInput.CharLimit = 120
	titleInput.Width = 50

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.CharLimit = 500
	descInput.Width = 50

	promptInput := textinput.New()
	promptInput.Placeholder = "Prompt (optional, sent when session starts)"
	promptInput.CharLimit = 2000
	promptInput.Width = 50

	return &TodoDialog{
		titleInput:  titleInput,
		descInput:   descInput,
		promptInput: promptInput,
		statusOptions: []session.TodoStatus{
			session.TodoStatusTodo,
			session.TodoStatusInProgress,
			session.TodoStatusInReview,
			session.TodoStatusDone,
			session.TodoStatusOrphaned,
		},
	}
}

// rebuildCols rebuilds the derived column structure from d.todos.
// Preserves the cursor position by todo ID where possible; otherwise clamps.
func (d *TodoDialog) rebuildCols() {
	// Snapshot current selected ID before rebuild
	var prevID string
	if t := d.SelectedTodo(); t != nil {
		prevID = t.ID
	}

	d.cols = buildColumns(d.todos)

	// Ensure selectedRow has one entry per column
	newRow := make([]int, len(d.cols))
	for i := range newRow {
		if i < len(d.selectedRow) {
			newRow[i] = d.selectedRow[i]
		}
	}
	d.selectedRow = newRow

	// Clamp each row cursor to valid range
	for i, col := range d.cols {
		if len(col.todos) == 0 {
			d.selectedRow[i] = 0
		} else if d.selectedRow[i] >= len(col.todos) {
			d.selectedRow[i] = len(col.todos) - 1
		}
	}

	// Clamp column cursor
	if len(d.cols) == 0 {
		d.selectedCol = 0
		return
	}
	if d.selectedCol >= len(d.cols) {
		d.selectedCol = len(d.cols) - 1
	}

	// Restore cursor to the same todo ID. Any found position is guaranteed to
	// be within bounds because buildColumns only includes todos from d.todos,
	// so no second clamp pass is needed.
	if prevID != "" {
		for colIdx, col := range d.cols {
			for rowIdx, t := range col.todos {
				if t.ID == prevID {
					d.selectedCol = colIdx
					d.selectedRow[colIdx] = rowIdx
					return
				}
			}
		}
	}
}

// IsVisible returns true if the dialog is open.
func (d *TodoDialog) IsVisible() bool { return d.visible }

// Show opens the dialog for the given project path with the given todos.
func (d *TodoDialog) Show(projectPath, groupPath, groupName string, todos []*session.Todo) {
	d.visible = true
	d.projectPath = projectPath
	d.groupPath = groupPath
	d.groupName = groupName
	d.todos = todos
	// Reset cursor to top-left. Clear cols so rebuildCols doesn't restore the
	// previous cursor position via ID lookup — Show() always opens fresh.
	d.cols = nil
	d.selectedCol = 0
	d.selectedRow = nil
	d.mode = todoModeKanban
	d.errorMsg = ""
	d.rebuildCols()
}

// Hide closes the dialog.
func (d *TodoDialog) Hide() { d.visible = false }

// SetSize updates dimensions.
func (d *TodoDialog) SetSize(w, h int) {
	d.width = w
	d.height = h
	inputWidth := w/2 - 6
	if inputWidth < 10 {
		inputWidth = 10
	}
	d.titleInput.Width = inputWidth
	d.descInput.Width = inputWidth
	d.promptInput.Width = inputWidth
}

// SetTodos replaces the current todo list (used after reloads).
func (d *TodoDialog) SetTodos(todos []*session.Todo) {
	d.todos = todos
	d.rebuildCols()
}

// SelectedTodo returns the currently selected todo, or nil.
func (d *TodoDialog) SelectedTodo() *session.Todo {
	if len(d.cols) == 0 || d.selectedCol >= len(d.cols) {
		return nil
	}
	col := d.cols[d.selectedCol]
	if len(col.todos) == 0 {
		return nil
	}
	if d.selectedCol >= len(d.selectedRow) {
		return nil
	}
	row := d.selectedRow[d.selectedCol]
	if row >= len(col.todos) {
		return nil
	}
	return col.todos[row]
}

// todoStatusIcon returns the icon string for a given status.
func todoStatusIcon(status session.TodoStatus) string {
	switch status {
	case session.TodoStatusTodo:
		return "○"
	case session.TodoStatusInProgress:
		return "●"
	case session.TodoStatusInReview:
		return "⟳"
	case session.TodoStatusDone:
		return "✓"
	case session.TodoStatusOrphaned:
		return "!"
	default:
		return "○"
	}
}

// todoStatusLabel returns display label for a status.
func todoStatusLabel(status session.TodoStatus) string {
	switch status {
	case session.TodoStatusTodo:
		return "todo"
	case session.TodoStatusInProgress:
		return "in progress"
	case session.TodoStatusInReview:
		return "in review"
	case session.TodoStatusDone:
		return "done"
	case session.TodoStatusOrphaned:
		return "orphaned"
	default:
		return string(status)
	}
}

// todoStatusStyle returns a lipgloss style for the icon/label.
func todoStatusStyle(status session.TodoStatus) lipgloss.Style {
	switch status {
	case session.TodoStatusTodo:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7a8a9a"))
	case session.TodoStatusInProgress:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff"))
	case session.TodoStatusInReview:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd75f"))
	case session.TodoStatusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#5faf5f"))
	case session.TodoStatusOrphaned:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	default:
		return lipgloss.NewStyle()
	}
}

// TodoAction is returned by HandleKey to signal what the caller should do.
type TodoAction int

const (
	TodoActionNone          TodoAction = iota
	TodoActionClose                    // user pressed esc/t in list mode
	TodoActionCreateSession            // create session+worktree from selected todo
	TodoActionSaveTodo                 // save new/edited todo (caller reads GetFormValues)
	TodoActionDeleteTodo               // delete selected todo
	TodoActionUpdateStatus             // update status of selected todo (caller reads GetPickedStatus)
	TodoActionMoveCardLeft             // shift+left: move selected card to previous status column
	TodoActionMoveCardRight            // shift+right: move selected card to next status column
)

// GetFormValues returns the title, description, prompt, editing ID, and status after TodoActionSaveTodo.
func (d *TodoDialog) GetFormValues() (title, description, prompt, editingID string, status session.TodoStatus) {
	return strings.TrimSpace(d.titleInput.Value()),
		strings.TrimSpace(d.descInput.Value()),
		strings.TrimSpace(d.promptInput.Value()),
		d.editingID,
		d.newTodoStatus
}

// GetPickedStatus returns the status chosen in the status picker.
func (d *TodoDialog) GetPickedStatus() session.TodoStatus {
	if d.statusCursor < len(d.statusOptions) {
		return d.statusOptions[d.statusCursor]
	}
	return session.TodoStatusTodo
}

// HandleKey processes a keypress and returns the action the caller should take.
func (d *TodoDialog) HandleKey(msg tea.KeyMsg) TodoAction {
	switch d.mode {
	case todoModeKanban:
		return d.handleKanbanKey(msg.String())
	case todoModeNew, todoModeEdit:
		return d.handleFormKey(msg)
	case todoModeStatus:
		return d.handleStatusKey(msg.String())
	}
	return TodoActionNone
}


func (d *TodoDialog) handleFormKey(msg tea.KeyMsg) TodoAction {
	key := msg.String()
	switch key {
	case "tab":
		switch d.formFocus {
		case 0:
			d.formFocus = 1
			d.titleInput.Blur()
			d.descInput.Focus()
		case 1:
			d.formFocus = 2
			d.descInput.Blur()
			d.promptInput.Focus()
		default:
			d.formFocus = 0
			d.promptInput.Blur()
			d.titleInput.Focus()
		}
	case "shift+tab":
		switch d.formFocus {
		case 0:
			d.formFocus = 2
			d.titleInput.Blur()
			d.promptInput.Focus()
		case 1:
			d.formFocus = 0
			d.descInput.Blur()
			d.titleInput.Focus()
		default:
			d.formFocus = 1
			d.promptInput.Blur()
			d.descInput.Focus()
		}
	case "enter":
		title := strings.TrimSpace(d.titleInput.Value())
		if title == "" {
			d.errorMsg = "Title is required"
			return TodoActionNone
		}
		return TodoActionSaveTodo
	case "esc":
		d.mode = todoModeKanban
		d.errorMsg = ""
	default:
		switch d.formFocus {
		case 0:
			d.titleInput, _ = d.titleInput.Update(msg)
		case 1:
			d.descInput, _ = d.descInput.Update(msg)
		case 2:
			d.promptInput, _ = d.promptInput.Update(msg)
		}
	}
	return TodoActionNone
}

func (d *TodoDialog) handleStatusKey(key string) TodoAction {
	switch key {
	case "up", "k":
		if d.statusCursor > 0 {
			d.statusCursor--
		}
	case "down", "j":
		if d.statusCursor < len(d.statusOptions)-1 {
			d.statusCursor++
		}
	case "enter":
		d.mode = todoModeKanban
		return TodoActionUpdateStatus
	case "esc":
		d.mode = todoModeKanban
	}
	return TodoActionNone
}

func (d *TodoDialog) openNewForm() {
	d.mode = todoModeNew
	d.editingID = ""
	d.titleInput.SetValue("")
	d.descInput.SetValue("")
	d.promptInput.SetValue("")
	d.formFocus = 0
	d.titleInput.Focus()
	d.descInput.Blur()
	d.promptInput.Blur()
	d.errorMsg = ""
	// Pre-select the focused column's status so new cards land in the right column
	if d.selectedCol < len(d.cols) {
		d.newTodoStatus = d.cols[d.selectedCol].status
	} else {
		d.newTodoStatus = session.TodoStatusTodo
	}
}

func (d *TodoDialog) openEditForm(t *session.Todo) {
	d.mode = todoModeEdit
	d.editingID = t.ID
	d.titleInput.SetValue(t.Title)
	d.descInput.SetValue(t.Description)
	d.promptInput.SetValue(t.Prompt)
	d.formFocus = 0
	d.titleInput.Focus()
	d.descInput.Blur()
	d.promptInput.Blur()
	d.errorMsg = ""
	d.newTodoStatus = t.Status // keeps GetFormValues coherent in edit flow
}

func (d *TodoDialog) openStatusPicker(t *session.Todo) {
	d.mode = todoModeStatus
	// Pre-select current status
	for i, s := range d.statusOptions {
		if s == t.Status {
			d.statusCursor = i
			break
		}
	}
}

// MoveCardTargetStatus returns the target status when moving the selected card
// by direction (+1 = right, -1 = left). Returns false if the move is a no-op
// (boundary or would move into the orphaned column).
func (d *TodoDialog) MoveCardTargetStatus(direction int) (session.TodoStatus, bool) {
	targetCol := d.selectedCol + direction
	if targetCol < 0 || targetCol >= len(d.cols) {
		return "", false
	}
	target := d.cols[targetCol].status
	if target == session.TodoStatusOrphaned {
		return "", false
	}
	return target, true
}

// ResetFormToList returns dialog to list mode (call after successful save/update).
func (d *TodoDialog) ResetFormToList() {
	d.mode = todoModeKanban
	d.errorMsg = ""
}

// View renders the dialog.
func (d *TodoDialog) View() string {
	if !d.visible {
		return ""
	}
	switch d.mode {
	case todoModeNew, todoModeEdit:
		return d.viewForm()
	case todoModeStatus:
		return d.viewStatusPicker()
	default:
		return d.viewKanban()
	}
}


func (d *TodoDialog) viewForm() string {
	title := "New Todo"
	if d.mode == todoModeEdit {
		title = "Edit Todo"
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5fd7ff")).
		Padding(1, 2).
		Width(60)

	header := lipgloss.NewStyle().Bold(true).Render(title)

	dimStyle    := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff"))
	titleLabel  := dimStyle.Render("Title:")
	descLabel   := dimStyle.Render("Description:")
	promptLabel := dimStyle.Render("Prompt (optional):")
	switch d.formFocus {
	case 0:
		titleLabel = activeStyle.Render("Title:")
	case 1:
		descLabel = activeStyle.Render("Description:")
	case 2:
		promptLabel = activeStyle.Render("Prompt (optional):")
	}

	var errLine string
	if d.errorMsg != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Render(d.errorMsg)
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("tab switch  enter save  esc cancel")

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s%s\n\n%s",
		header,
		titleLabel, d.titleInput.View(),
		descLabel, d.descInput.View(),
		promptLabel, d.promptInput.View(),
		errLine, hint,
	)

	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
		borderStyle.Render(content))
}

func (d *TodoDialog) viewStatusPicker() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5fd7ff")).
		Padding(1, 2).
		Width(30)

	header := lipgloss.NewStyle().Bold(true).Render("Change Status")

	var rows []string
	for i, s := range d.statusOptions {
		st := todoStatusStyle(s)
		line := fmt.Sprintf(" %s %s", st.Render(todoStatusIcon(s)), todoStatusLabel(s))
		if i == d.statusCursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#2a3a4a")).
				Render(line)
		}
		rows = append(rows, line)
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("↑↓ move  enter select  esc cancel")
	content := header + "\n\n" + strings.Join(rows, "\n") + "\n\n" + hint

	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
		borderStyle.Render(content))
}

// TodoBranchName converts a todo title to a git branch name.
func TodoBranchName(title string) string {
	lower := strings.ToLower(title)
	return git.SanitizeBranchName(lower)
}

func (d *TodoDialog) handleKanbanKey(key string) TodoAction {
	switch key {
	case "left", "h":
		if d.selectedCol > 0 {
			d.selectedCol--
		}
	case "right", "l":
		if d.selectedCol < len(d.cols)-1 {
			d.selectedCol++
		}
	case "up", "k":
		if d.selectedCol < len(d.selectedRow) && d.selectedRow[d.selectedCol] > 0 {
			d.selectedRow[d.selectedCol]--
		}
	case "down", "j":
		if d.selectedCol < len(d.cols) && d.selectedCol < len(d.selectedRow) {
			col := d.cols[d.selectedCol]
			if d.selectedRow[d.selectedCol] < len(col.todos)-1 {
				d.selectedRow[d.selectedCol]++
			}
		}
	case "shift+left":
		// No card selected or already at boundary; no-op.
		if d.SelectedTodo() != nil {
			if _, ok := d.MoveCardTargetStatus(-1); ok {
				return TodoActionMoveCardLeft
			}
		}
	case "shift+right":
		// No card selected or already at boundary; no-op.
		if d.SelectedTodo() != nil {
			if _, ok := d.MoveCardTargetStatus(1); ok {
				return TodoActionMoveCardRight
			}
		}
	case "n":
		d.openNewForm()
	case "e":
		if t := d.SelectedTodo(); t != nil {
			d.openEditForm(t)
		}
	case "d":
		if d.SelectedTodo() != nil {
			return TodoActionDeleteTodo
		}
	case "s":
		if t := d.SelectedTodo(); t != nil {
			d.openStatusPicker(t)
		}
	case "enter":
		if d.SelectedTodo() == nil {
			return TodoActionNone
		}
		return TodoActionCreateSession
	case "esc", "t":
		return TodoActionClose
	}
	return TodoActionNone
}

// wordWrapText wraps text at word boundaries to fit within width, returning at
// most maxLines lines. If content is cut, the last line ends with "…".
// Uses rune-based width so multi-byte characters are counted correctly.
func wordWrapText(text string, width, maxLines int) string {
	if text == "" || width <= 0 || maxLines <= 0 {
		return ""
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	line := ""
	for _, w := range words {
		switch {
		case line == "":
			line = w
		case len([]rune(line))+1+len([]rune(w)) <= width:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	// Truncate: keep first maxLines-1 lines, truncate the maxLines-th line.
	result := make([]string, maxLines)
	copy(result, lines[:maxLines-1])
	last := []rune(lines[maxLines-1])
	if len(last) > width-1 {
		last = last[:width-1]
	}
	result[maxLines-1] = string(last) + "…"
	return strings.Join(result, "\n")
}

// renderDetailPanel renders the description box for the currently selected todo.
// innerW is the usable content width (same value used for the kanban columns).
// Returns an empty string if no todo is selected.
func (d *TodoDialog) renderDetailPanel(innerW int) string {
	t := d.SelectedTodo()
	if t == nil {
		return ""
	}

	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5a6a7a")).
		Render("description")

	textWidth := innerW - 4
	if textWidth < 10 {
		textWidth = 10
	}

	var body string
	if t.Description == "" {
		body = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3a4a5a")).
			Render("no description")
	} else {
		wrapped := wordWrapText(t.Description, textWidth, 3)
		body = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0ccd8")).
			Render(wrapped)
	}

	content := label + "\n" + body

	if t.Prompt != "" {
		promptLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("⌨ prompt")
		promptPreview := wordWrapText(t.Prompt, textWidth, 2)
		promptBody := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0ccd8")).Italic(true).Render(promptPreview)
		content += "\n\n" + promptLabel + "\n" + promptBody
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3a4a5a")).
		Padding(0, 1).
		Width(innerW - 2).
		Render(content)
}

func (d *TodoDialog) viewKanban() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5fd7ff")).
		Padding(0, 1).
		Width(d.width - 4)

	projectName := d.projectPath
	if idx := strings.LastIndex(projectName, "/"); idx >= 0 {
		projectName = projectName[idx+1:]
	}
	header := lipgloss.NewStyle().Bold(true).Render(projectName)

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render(
		"←/→ col  ↑/↓ card  n new  enter open  s status  e edit  d delete  shift+←/→ move  esc close",
	)

	// Empty board
	if len(d.todos) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render(
			"No todos yet — press n to create one",
		)
		content := header + "\n\n" + empty + "\n\n" + hint
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
			borderStyle.Render(content))
	}

	numCols := len(d.cols)
	innerW := d.width - 6 // border(2) + padding(2) + margin(2)
	if innerW < numCols*8 {
		innerW = numCols * 8
	}
	colW := (innerW - (numCols - 1)) / numCols
	if colW < 8 {
		colW = 8
	}

	colViews := make([]string, numCols)
	for i, col := range d.cols {
		colViews[i] = d.renderKanbanColumn(i, col, colW)
	}
	board := lipgloss.JoinHorizontal(lipgloss.Top, colViews...)

	detail := d.renderDetailPanel(innerW)
	var content string
	if detail != "" {
		content = header + "\n\n" + board + "\n\n" + detail + "\n" + hint
	} else {
		content = header + "\n\n" + board + "\n\n" + hint
	}
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
		borderStyle.Render(content))
}

func (d *TodoDialog) renderKanbanColumn(colIdx int, col kanbanColumn, width int) string {
	isFocused := colIdx == d.selectedCol

	headerText := fmt.Sprintf("%s (%d)", col.label, len(col.todos))
	var header string
	if isFocused {
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5fd7ff")).
			Bold(true).
			Render(headerText)
	} else {
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7a8a9a")).
			Render(headerText)
	}

	underlineColor := "#5fd7ff"
	if !isFocused {
		underlineColor = "#3a4a5a"
	}
	underline := lipgloss.NewStyle().
		Foreground(lipgloss.Color(underlineColor)).
		Render(strings.Repeat("─", width))

	lines := []string{header, underline}

	if len(col.todos) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3a4a5a")).
			Render("  (empty)"))
	}

	selectedRow := 0
	if colIdx < len(d.selectedRow) {
		selectedRow = d.selectedRow[colIdx]
	}

	for rowIdx, t := range col.todos {
		isSelected := isFocused && rowIdx == selectedRow
		lines = append(lines, d.renderKanbanCard(t, isSelected, isFocused, width))
	}

	colContent := strings.Join(lines, "\n")
	// +1 char right gap between columns
	return lipgloss.NewStyle().Width(width + 1).Render(colContent)
}

func (d *TodoDialog) renderKanbanCard(t *session.Todo, isSelected, colFocused bool, width int) string {
	icon := todoStatusIcon(t.Status)
	sessionMark := ""
	if t.SessionID != "" {
		sessionMark = " ⬡"
	}

	// Available title width: selector(1) + icon(1) + space(1) + title + sessionMark
	titleWidth := width - 3 - lipgloss.Width(sessionMark)
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := t.Title
	runes := []rune(title)
	if len(runes) > titleWidth {
		if titleWidth > 3 {
			title = string(runes[:titleWidth-3]) + "..."
		} else {
			title = string(runes[:titleWidth])
		}
	}

	selector := " "
	if isSelected {
		selector = "▌"
	}

	switch {
	case isSelected:
		styledIcon := todoStatusStyle(t.Status).Render(icon)
		line := fmt.Sprintf("%s%s %s%s", selector, styledIcon, title, sessionMark)
		return lipgloss.NewStyle().
			Background(lipgloss.Color("#2a3a4a")).
			Foreground(lipgloss.Color("#ffffff")).
			Width(width).
			Render(line)
	case !colFocused:
		line := fmt.Sprintf("%s%s %s%s", selector, icon, title, sessionMark)
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4a5a6a")).
			Width(width).
			Render(line)
	default:
		styledIcon := todoStatusStyle(t.Status).Render(icon)
		line := fmt.Sprintf("%s%s %s%s", selector, styledIcon, title, sessionMark)
		return lipgloss.NewStyle().Width(width).Render(line)
	}
}
