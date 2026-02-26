package ui

import (
	"fmt"
	"strings"

	"ghe.spotify.net/mnicholson/hangar/internal/git"
	"ghe.spotify.net/mnicholson/hangar/internal/session"
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
	formFocus     int               // 0=title, 1=desc
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

	return &TodoDialog{
		titleInput: titleInput,
		descInput:  descInput,
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

// GetFormValues returns the title, description, editing ID, and status after TodoActionSaveTodo.
func (d *TodoDialog) GetFormValues() (title, description, editingID string, status session.TodoStatus) {
	return strings.TrimSpace(d.titleInput.Value()),
		strings.TrimSpace(d.descInput.Value()),
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

func (d *TodoDialog) handleListKey(key string) TodoAction {
	switch key {
	case "up", "k":
		if d.selectedCol < len(d.selectedRow) && d.selectedRow[d.selectedCol] > 0 {
			d.selectedRow[d.selectedCol]--
		}
	case "down", "j":
		if d.selectedCol < len(d.cols) && d.selectedRow[d.selectedCol] < len(d.cols[d.selectedCol].todos)-1 {
			d.selectedRow[d.selectedCol]++
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
		t := d.SelectedTodo()
		if t == nil {
			return TodoActionNone
		}
		// todo status with no session: create new session+worktree
		// any other status or linked session: attach/handle
		return TodoActionCreateSession
	case "esc", "t":
		return TodoActionClose
	}
	return TodoActionNone
}

func (d *TodoDialog) handleFormKey(msg tea.KeyMsg) TodoAction {
	key := msg.String()
	switch key {
	case "tab", "shift+tab":
		if d.formFocus == 0 {
			d.formFocus = 1
			d.titleInput.Blur()
			d.descInput.Focus()
		} else {
			d.formFocus = 0
			d.descInput.Blur()
			d.titleInput.Focus()
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
		// Pass the real tea.KeyMsg to preserve Type (backspace, arrows, ctrl+w, etc.)
		if d.formFocus == 0 {
			d.titleInput, _ = d.titleInput.Update(msg)
		} else {
			d.descInput, _ = d.descInput.Update(msg)
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
	d.formFocus = 0
	d.titleInput.Focus()
	d.descInput.Blur()
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
	d.formFocus = 0
	d.titleInput.Focus()
	d.descInput.Blur()
	d.errorMsg = ""
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

func (d *TodoDialog) viewList() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5fd7ff")).
		Padding(0, 1).
		Width(d.width - 4).
		MaxHeight(d.height - 4)

	projectName := d.projectPath
	if idx := strings.LastIndex(projectName, "/"); idx >= 0 {
		projectName = projectName[idx+1:]
	}

	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%s  [%d todos]", projectName, len(d.todos)),
	)

	var rows []string
	if len(d.todos) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#7a8a9a")).Render("  No todos yet. Press n to add one."))
	}

	innerWidth := d.width - 10
	if innerWidth < 20 {
		innerWidth = 20
	}
	selectedTodo := d.SelectedTodo()
	for _, t := range d.todos {
		statusSt := todoStatusStyle(t.Status)
		icon := statusSt.Render(todoStatusIcon(t.Status))
		label := statusSt.Render(todoStatusLabel(t.Status))

		titleCol := t.Title
		truncAt := innerWidth - 18
		if truncAt > 0 && len(titleCol) > innerWidth-15 {
			titleCol = titleCol[:truncAt] + "..."
		}

		gap := innerWidth - len(titleCol) - len(todoStatusLabel(t.Status)) - 4
		if gap < 1 {
			gap = 1
		}
		line := fmt.Sprintf(" %s %s%s%s", icon, titleCol, strings.Repeat(" ", gap), label)

		isSelected := selectedTodo != nil && t.ID == selectedTodo.ID
		if isSelected {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#2a3a4a")).
				Foreground(lipgloss.Color("#ffffff")).
				Render(line)
		}
		rows = append(rows, line)

		// For the selected item, show description and session hint
		if isSelected {
			if t.Description != "" {
				desc := t.Description
				maxDescWidth := innerWidth - 5
				if maxDescWidth > 0 && len(desc) > maxDescWidth {
					desc = desc[:maxDescWidth-3] + "..."
				}
				rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#8a9aaa")).Render("   "+desc))
			}
			if t.SessionID != "" {
				rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("   └─ session linked"))
			}
		}
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render(
		"n new  enter open  s status  e edit  d delete  esc close",
	)

	content := strings.Join(append([]string{header, ""}, append(rows, "", hint)...), "\n")
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
		borderStyle.Render(content))
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

	titleLabel := "Title:"
	descLabel := "Description:"
	if d.formFocus == 0 {
		titleLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff")).Render("Title:")
	} else {
		descLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff")).Render("Description:")
	}

	var errLine string
	if d.errorMsg != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Render(d.errorMsg)
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("tab switch  enter save  esc cancel")

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s%s\n\n%s",
		header,
		titleLabel, d.titleInput.View(),
		descLabel, d.descInput.View(),
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
		if d.selectedCol < len(d.cols) {
			col := d.cols[d.selectedCol]
			if d.selectedRow[d.selectedCol] < len(col.todos)-1 {
				d.selectedRow[d.selectedCol]++
			}
		}
	case "shift+left":
		if d.SelectedTodo() != nil {
			if _, ok := d.MoveCardTargetStatus(-1); ok {
				return TodoActionMoveCardLeft
			}
		}
	case "shift+right":
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

func (d *TodoDialog) viewKanban() string {
	return d.viewList() // temporary stub, replaced in Task 5
}
