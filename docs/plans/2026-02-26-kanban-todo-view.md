# Kanban Todo View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the linear todo list in `TodoDialog` with a kanban board where columns represent status.

**Architecture:** Swap `todoModeList` for `todoModeKanban` inside the existing `TodoDialog`. Replace the single `cursor int` with a 2D `selectedCol`/`selectedRow[]int` cursor. Derive `[]kanbanColumn` from the todo slice at render/key time — no new storage. Add `TodoActionMoveCardLeft`/`TodoActionMoveCardRight` and handle them in `home.go` exactly like `TodoActionUpdateStatus`.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), lipgloss (`github.com/charmbracelet/lipgloss`), standard `testing` package.

---

### Task 1: Add `kanbanColumn` type and `buildColumns()` with unit tests

**Files:**
- Modify: `internal/ui/todo_dialog.go` (after the imports, before `todoDialogMode`)
- Create: `internal/ui/todo_dialog_test.go`

**Step 1: Write failing tests first**

Create `internal/ui/todo_dialog_test.go`:

```go
package ui

import (
	"testing"

	"ghe.spotify.net/mnicholson/hangar/internal/session"
)

func makeTodo(id string, status session.TodoStatus) *session.Todo {
	return &session.Todo{ID: id, Title: id, Status: status}
}

func TestBuildColumns_AlwaysFourMainCols(t *testing.T) {
	cols := buildColumns(nil)
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}
	if cols[0].status != session.TodoStatusTodo {
		t.Errorf("col 0 should be Todo, got %s", cols[0].status)
	}
	if cols[1].status != session.TodoStatusInProgress {
		t.Errorf("col 1 should be InProgress, got %s", cols[1].status)
	}
	if cols[2].status != session.TodoStatusInReview {
		t.Errorf("col 2 should be InReview, got %s", cols[2].status)
	}
	if cols[3].status != session.TodoStatusDone {
		t.Errorf("col 3 should be Done, got %s", cols[3].status)
	}
}

func TestBuildColumns_OrphanedColHiddenWhenNone(t *testing.T) {
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusDone),
	}
	cols := buildColumns(todos)
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns (no orphaned), got %d", len(cols))
	}
}

func TestBuildColumns_OrphanedColShownWhenPresent(t *testing.T) {
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusOrphaned),
	}
	cols := buildColumns(todos)
	if len(cols) != 5 {
		t.Fatalf("expected 5 columns (with orphaned), got %d", len(cols))
	}
	if cols[4].status != session.TodoStatusOrphaned {
		t.Errorf("col 4 should be Orphaned, got %s", cols[4].status)
	}
}

func TestBuildColumns_TodosDistributedCorrectly(t *testing.T) {
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusTodo),
		makeTodo("c", session.TodoStatusInProgress),
		makeTodo("d", session.TodoStatusDone),
	}
	cols := buildColumns(todos)
	if len(cols[0].todos) != 2 {
		t.Errorf("Todo column should have 2 items, got %d", len(cols[0].todos))
	}
	if len(cols[1].todos) != 1 {
		t.Errorf("InProgress column should have 1 item, got %d", len(cols[1].todos))
	}
	if len(cols[2].todos) != 0 {
		t.Errorf("InReview column should be empty, got %d", len(cols[2].todos))
	}
	if len(cols[3].todos) != 1 {
		t.Errorf("Done column should have 1 item, got %d", len(cols[3].todos))
	}
}
```

**Step 2: Run tests to confirm they fail**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/kanban-todo-view
go test ./internal/ui/... -run TestBuildColumns -v
```

Expected: FAIL — `buildColumns` undefined.

**Step 3: Add `kanbanColumn` type and `buildColumns()` to `todo_dialog.go`**

Insert after line 12 (end of imports), before line 14 (`// todoDialogMode`):

```go
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
		} else if t.Status == session.TodoStatusOrphaned {
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
```

**Step 4: Run tests to confirm they pass**

```bash
go test ./internal/ui/... -run TestBuildColumns -v
```

Expected: PASS (4 tests).

**Step 5: Commit**

```bash
git add internal/ui/todo_dialog.go internal/ui/todo_dialog_test.go
git commit -m "feat(kanban): add kanbanColumn type and buildColumns()"
```

---

### Task 2: Update `TodoDialog` struct with 2D cursor state

**Files:**
- Modify: `internal/ui/todo_dialog.go`

**Step 1: Write failing test for cursor state**

Add to `internal/ui/todo_dialog_test.go`:

```go
func TestTodoDialog_SelectedTodo_EmptyBoard(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	if d.SelectedTodo() != nil {
		t.Error("expected nil SelectedTodo on empty board")
	}
}

func TestTodoDialog_SelectedTodo_FirstTodo(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("x", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)
	got := d.SelectedTodo()
	if got == nil || got.ID != "x" {
		t.Errorf("expected todo x, got %v", got)
	}
}

func TestTodoDialog_SetTodos_PreservesCursorByID(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusInProgress),
	}
	d.Show("/proj", "", "", todos)
	// Move cursor to the InProgress column (col 1)
	d.selectedCol = 1
	// Now reload todos — b should still be selected
	d.SetTodos(todos)
	got := d.SelectedTodo()
	if got == nil || got.ID != "b" {
		t.Errorf("expected cursor to stay on b, got %v", got)
	}
}
```

**Step 2: Run tests to confirm they fail**

```bash
go test ./internal/ui/... -run TestTodoDialog_SelectedTodo -v
go test ./internal/ui/... -run TestTodoDialog_SetTodos -v
```

Expected: FAIL — `selectedCol` field undefined, `SelectedTodo()` uses old cursor.

**Step 3: Update `TodoDialog` struct in `todo_dialog.go` (lines 25–46)**

Replace the struct definition with:

```go
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
	formFocus     int              // 0=title, 1=desc
	editingID     string           // non-empty when editing
	newTodoStatus session.TodoStatus // status pre-selected from focused column when pressing n

	// status picker
	statusOptions []session.TodoStatus
	statusCursor  int
}
```

**Step 4: Add `rebuildCols()` method after `NewTodoDialog()` (after line 71)**

```go
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

	// Restore cursor to the same todo ID
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
```

**Step 5: Update `Show()` (lines 77–86) to init kanban state**

```go
func (d *TodoDialog) Show(projectPath, groupPath, groupName string, todos []*session.Todo) {
	d.visible = true
	d.projectPath = projectPath
	d.groupPath = groupPath
	d.groupName = groupName
	d.todos = todos
	d.selectedCol = 0
	d.selectedRow = nil // rebuildCols initialises this
	d.mode = todoModeKanban
	d.errorMsg = ""
	d.rebuildCols()
}
```

(Note: `todoModeKanban` is renamed in Task 3 — if doing tasks in order the build will fail until Task 3 is done. So do Tasks 2 and 3 together before running tests.)

**Step 6: Update `SetTodos()` (lines 104–113)**

```go
func (d *TodoDialog) SetTodos(todos []*session.Todo) {
	d.todos = todos
	d.rebuildCols()
}
```

**Step 7: Update `SelectedTodo()` (lines 116–121)**

```go
func (d *TodoDialog) SelectedTodo() *session.Todo {
	if len(d.cols) == 0 || d.selectedCol >= len(d.cols) {
		return nil
	}
	col := d.cols[d.selectedCol]
	if len(col.todos) == 0 {
		return nil
	}
	row := d.selectedRow[d.selectedCol]
	if row >= len(col.todos) {
		return nil
	}
	return col.todos[row]
}
```

**Step 8: Confirm build passes (Task 3 must also be done first — do not run tests yet)**

---

### Task 3: Rename `todoModeList` → `todoModeKanban`, update all references

**Files:**
- Modify: `internal/ui/todo_dialog.go`

These are mechanical find-and-replace changes.

**Step 1: Update mode constant (line 18)**

Change:
```go
todoModeList   todoDialogMode = iota
```
To:
```go
todoModeKanban todoDialogMode = iota
```

**Step 2: Update `HandleKey()` (line 207)**

Change:
```go
case todoModeList:
    return d.handleListKey(msg.String())
```
To:
```go
case todoModeKanban:
    return d.handleKanbanKey(msg.String())
```

(`handleKanbanKey` is written in Task 4. The method `handleListKey` can stay temporarily until replaced.)

**Step 3: Update `handleFormKey()` — esc goes back to kanban (line 276)**

Change:
```go
case "esc":
    d.mode = todoModeList
```
To:
```go
case "esc":
    d.mode = todoModeKanban
```

**Step 4: Update `handleStatusKey()` — enter and esc go back to kanban (lines 300, 303)**

Change:
```go
case "enter":
    d.mode = todoModeList
    return TodoActionUpdateStatus
case "esc":
    d.mode = todoModeList
```
To:
```go
case "enter":
    d.mode = todoModeKanban
    return TodoActionUpdateStatus
case "esc":
    d.mode = todoModeKanban
```

**Step 5: Update `ResetFormToList()` (line 343)**

Change:
```go
d.mode = todoModeList
```
To:
```go
d.mode = todoModeKanban
```

**Step 6: Update `View()` switch (line 357)**

Change:
```go
default:
    return d.viewList()
```
To:
```go
default:
    return d.viewKanban()
```

(`viewKanban` will be written in Task 5.)

**Step 7: Confirm build compiles (with stubs for `handleKanbanKey` and `viewKanban`)**

Add temporary stubs at end of file to unblock compilation:

```go
func (d *TodoDialog) handleKanbanKey(key string) TodoAction {
	return d.handleListKey(key) // temporary stub, replaced in Task 4
}

func (d *TodoDialog) viewKanban() string {
	return d.viewList() // temporary stub, replaced in Task 5
}
```

**Step 8: Build and run all existing tests**

```bash
go build ./...
go test ./internal/ui/... -v 2>&1 | tail -30
```

Expected: build passes; prior tests still pass (2 pre-existing failures are OK).

**Step 9: Commit**

```bash
git add internal/ui/todo_dialog.go internal/ui/todo_dialog_test.go
git commit -m "feat(kanban): restructure TodoDialog with 2D cursor and todoModeKanban"
```

---

### Task 4: Add move-card actions, update `GetFormValues()`, write `handleKanbanKey()`

**Files:**
- Modify: `internal/ui/todo_dialog.go`

**Step 1: Write failing tests**

Add to `internal/ui/todo_dialog_test.go`:

```go
func TestTodoDialog_HandleKanban_LeftRight(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusInProgress),
	}
	d.Show("/proj", "", "", todos)

	// Start at col 0 (Todo)
	if d.selectedCol != 0 {
		t.Fatalf("expected col 0, got %d", d.selectedCol)
	}

	// Press right → move to InProgress col
	d.HandleKey(keyMsg("right"))
	if d.selectedCol != 1 {
		t.Errorf("expected col 1 after right, got %d", d.selectedCol)
	}

	// Press left → back to Todo col
	d.HandleKey(keyMsg("left"))
	if d.selectedCol != 0 {
		t.Errorf("expected col 0 after left, got %d", d.selectedCol)
	}

	// Press left at col 0 → stays at 0 (no wrap)
	d.HandleKey(keyMsg("left"))
	if d.selectedCol != 0 {
		t.Errorf("expected col 0 (no wrap), got %d", d.selectedCol)
	}
}

func TestTodoDialog_HandleKanban_NewPreSelectsColumnStatus(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	// Move to InProgress column (col 1)
	d.selectedCol = 1
	d.HandleKey(keyMsg("n"))
	_, _, _, status := d.GetFormValues()
	if status != session.TodoStatusInProgress {
		t.Errorf("expected InProgress status for new todo in col 1, got %s", status)
	}
}

func TestTodoDialog_MoveCardTargetStatus(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)

	// At col 0 (Todo), moving right → InProgress
	status, ok := d.MoveCardTargetStatus(1)
	if !ok || status != session.TodoStatusInProgress {
		t.Errorf("expected InProgress, got %s ok=%v", status, ok)
	}

	// At col 0 (Todo), moving left → no-op
	_, ok = d.MoveCardTargetStatus(-1)
	if ok {
		t.Error("expected no-op when moving left from Todo column")
	}
}

// keyMsg is a test helper that creates a tea.KeyMsg from a string.
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
```

**Step 2: Run tests to confirm they fail**

```bash
go test ./internal/ui/... -run "TestTodoDialog_HandleKanban|TestTodoDialog_MoveCard" -v
```

Expected: FAIL — `GetFormValues()` wrong signature, `MoveCardTargetStatus` undefined.

**Step 3: Add new `TodoAction` constants (after `TodoActionUpdateStatus` on line 186)**

```go
TodoActionMoveCardLeft  // shift+left: move selected card to previous status column
TodoActionMoveCardRight // shift+right: move selected card to next status column
```

**Step 4: Update `GetFormValues()` to return status (line 190–194)**

Change:
```go
func (d *TodoDialog) GetFormValues() (title, description, editingID string) {
	return strings.TrimSpace(d.titleInput.Value()),
		strings.TrimSpace(d.descInput.Value()),
		d.editingID
}
```
To:
```go
func (d *TodoDialog) GetFormValues() (title, description, editingID string, status session.TodoStatus) {
	return strings.TrimSpace(d.titleInput.Value()),
		strings.TrimSpace(d.descInput.Value()),
		d.editingID,
		d.newTodoStatus
}
```

**Step 5: Update `openNewForm()` to capture column status (line 308–317)**

Change:
```go
func (d *TodoDialog) openNewForm() {
	d.mode = todoModeNew
	d.editingID = ""
	d.titleInput.SetValue("")
	d.descInput.SetValue("")
	d.formFocus = 0
	d.titleInput.Focus()
	d.descInput.Blur()
	d.errorMsg = ""
}
```
To:
```go
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
```

**Step 6: Add `MoveCardTargetStatus()` helper**

Add after `openStatusPicker()`:

```go
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
```

**Step 7: Write `handleKanbanKey()`, replacing the temporary stub**

Replace the temporary `handleKanbanKey` stub with:

```go
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
```

**Step 8: Fix compile error in `home.go` caused by `GetFormValues()` signature change**

In `internal/ui/home.go` line 8953, change:
```go
title, desc, editingID := h.todoDialog.GetFormValues()
```
To:
```go
title, desc, editingID, newStatus := h.todoDialog.GetFormValues()
```

Then on line 8956, change:
```go
todo := session.NewTodo(title, desc, projectPath)
```
To:
```go
todo := session.NewTodo(title, desc, projectPath)
todo.Status = newStatus
```

**Step 9: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... -run "TestBuildColumns|TestTodoDialog" -v
```

Expected: all new tests pass. Two pre-existing failures are OK.

**Step 10: Commit**

```bash
git add internal/ui/todo_dialog.go internal/ui/todo_dialog_test.go internal/ui/home.go
git commit -m "feat(kanban): add handleKanbanKey, move-card actions, status pre-selection"
```

---

### Task 5: Write `viewKanban()` — replace the stub with real rendering

**Files:**
- Modify: `internal/ui/todo_dialog.go`

**Step 1: Write rendering smoke tests**

Add to `internal/ui/todo_dialog_test.go`:

```go
func TestTodoDialog_ViewKanban_ShowsColumnHeaders(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	todos := []*session.Todo{
		makeTodo("fix bug", session.TodoStatusTodo),
		makeTodo("auth work", session.TodoStatusInProgress),
	}
	d.Show("/myproject", "", "", todos)

	view := d.View()
	if !strings.Contains(view, "todo") {
		t.Errorf("expected 'todo' column header in view:\n%s", view)
	}
	if !strings.Contains(view, "in progress") {
		t.Errorf("expected 'in progress' column header in view:\n%s", view)
	}
	if !strings.Contains(view, "in review") {
		t.Errorf("expected 'in review' column header in view:\n%s", view)
	}
	if !strings.Contains(view, "done") {
		t.Errorf("expected 'done' column header in view:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_ShowsTodoTitle(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	todos := []*session.Todo{makeTodo("fix bug", session.TodoStatusTodo)}
	d.Show("/myproject", "", "", todos)

	view := d.View()
	if !strings.Contains(view, "fix bug") {
		t.Errorf("expected todo title in view:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_EmptyBoardMessage(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	d.Show("/myproject", "", "", nil)

	view := d.View()
	if !strings.Contains(view, "No todos yet") {
		t.Errorf("expected empty board message in view:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_OrphanedColHiddenWhenEmpty(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	d.Show("/myproject", "", "", nil)

	view := d.View()
	if strings.Contains(view, "orphaned") {
		t.Errorf("orphaned column should be hidden when no orphaned todos:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_SessionLinkIndicator(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	t1 := makeTodo("linked task", session.TodoStatusInProgress)
	t1.SessionID = "sess-123"
	d.Show("/myproject", "", "", []*session.Todo{t1})

	view := d.View()
	if !strings.Contains(view, "⬡") {
		t.Errorf("expected session link indicator ⬡ in view:\n%s", view)
	}
}
```

**Step 2: Run tests to confirm they fail**

```bash
go test ./internal/ui/... -run TestTodoDialog_ViewKanban -v
```

Expected: FAIL — `viewKanban` is still the stub that calls `viewList`.

**Step 3: Replace the `viewKanban` stub with real implementation**

Delete the stub. Replace with:

```go
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
	// Each column gets equal width; 1-char gap between columns accounted for
	colW := (innerW - (numCols - 1)) / numCols
	if colW < 8 {
		colW = 8
	}

	colViews := make([]string, numCols)
	for i, col := range d.cols {
		colViews[i] = d.renderKanbanColumn(i, col, colW)
	}
	board := lipgloss.JoinHorizontal(lipgloss.Top, colViews...)

	content := header + "\n\n" + board + "\n\n" + hint
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center,
		borderStyle.Render(content))
}

func (d *TodoDialog) renderKanbanColumn(colIdx int, col kanbanColumn, width int) string {
	isFocused := colIdx == d.selectedCol

	// Header label with count
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

	// Underline
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
	// Add 1 char right gap between columns (except last)
	return lipgloss.NewStyle().Width(width + 1).Render(colContent)
}

func (d *TodoDialog) renderKanbanCard(t *session.Todo, isSelected, colFocused bool, width int) string {
	icon := todoStatusIcon(t.Status)
	sessionMark := ""
	if t.SessionID != "" {
		sessionMark = " ⬡"
	}

	// Available title width: width - selector(1) - icon(1) - spaces(2) - sessionMark
	titleWidth := width - 4 - len(sessionMark)
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := t.Title
	if len(title) > titleWidth {
		if titleWidth > 3 {
			title = title[:titleWidth-3] + "..."
		} else {
			title = title[:titleWidth]
		}
	}

	selector := " "
	if isSelected {
		selector = "▌"
	}

	line := fmt.Sprintf("%s%s %s%s", selector, icon, title, sessionMark)

	switch {
	case isSelected:
		st := todoStatusStyle(t.Status)
		styledIcon := st.Render(icon)
		line = fmt.Sprintf("%s%s %s%s", selector, styledIcon, title, sessionMark)
		return lipgloss.NewStyle().
			Background(lipgloss.Color("#2a3a4a")).
			Foreground(lipgloss.Color("#ffffff")).
			Width(width).
			Render(line)
	case !colFocused:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4a5a6a")).
			Width(width).
			Render(line)
	default:
		st := todoStatusStyle(t.Status)
		styledIcon := st.Render(icon)
		line = fmt.Sprintf("%s%s %s%s", selector, styledIcon, title, sessionMark)
		return lipgloss.NewStyle().Width(width).Render(line)
	}
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestTodoDialog_ViewKanban -v
```

Expected: all 5 new view tests pass.

**Step 5: Run full test suite**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | tail -20
```

Expected: all new tests pass; 2 pre-existing failures unchanged.

**Step 6: Commit**

```bash
git add internal/ui/todo_dialog.go internal/ui/todo_dialog_test.go
git commit -m "feat(kanban): implement viewKanban with column rendering"
```

---

### Task 6: Update `home.go` to handle move-card actions

**Files:**
- Modify: `internal/ui/home.go`

The `GetFormValues()` signature fix was already done in Task 4, Step 8. This task adds the two new action cases.

**Step 1: Write test**

Add to `internal/ui/todo_dialog_test.go`:

```go
func TestTodoDialog_HandleKey_ShiftRight_ReturnsMoveCardRight(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)

	// shift+right on a Todo card → should return MoveCardRight
	action := d.HandleKey(tea.KeyMsg{Type: tea.KeyShiftRight})
	if action != TodoActionMoveCardRight {
		t.Errorf("expected TodoActionMoveCardRight, got %v", action)
	}
}

func TestTodoDialog_HandleKey_ShiftLeft_NoOpAtBoundary(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)

	// shift+left at leftmost column → no-op
	action := d.HandleKey(tea.KeyMsg{Type: tea.KeyShiftLeft})
	if action != TodoActionNone {
		t.Errorf("expected TodoActionNone at left boundary, got %v", action)
	}
}
```

**Step 2: Run new tests**

```bash
go test ./internal/ui/... -run "TestTodoDialog_HandleKey_Shift" -v
```

Expected: PASS (the logic was already written in Task 4).

**Step 3: Add `TodoActionMoveCardLeft` and `TodoActionMoveCardRight` cases to `handleTodoDialogKey()` in `home.go`**

In `internal/ui/home.go`, locate `handleTodoDialogKey()` (around line 8946). After the `TodoActionUpdateStatus` case (ends around line 9016), add:

```go
case TodoActionMoveCardLeft:
	todo := h.todoDialog.SelectedTodo()
	if todo != nil {
		if targetStatus, ok := h.todoDialog.MoveCardTargetStatus(-1); ok {
			if err := h.storage.UpdateTodoStatus(todo.ID, targetStatus, todo.SessionID); err != nil {
				h.setError(fmt.Errorf("move card: %w", err))
				return h, nil
			}
			todos, err := h.storage.LoadTodos(h.todoDialog.projectPath)
			if err != nil {
				h.setError(fmt.Errorf("reload todos: %w", err))
				return h, nil
			}
			h.todoDialog.SetTodos(todos)
		}
	}

case TodoActionMoveCardRight:
	todo := h.todoDialog.SelectedTodo()
	if todo != nil {
		if targetStatus, ok := h.todoDialog.MoveCardTargetStatus(1); ok {
			if err := h.storage.UpdateTodoStatus(todo.ID, targetStatus, todo.SessionID); err != nil {
				h.setError(fmt.Errorf("move card: %w", err))
				return h, nil
			}
			todos, err := h.storage.LoadTodos(h.todoDialog.projectPath)
			if err != nil {
				h.setError(fmt.Errorf("reload todos: %w", err))
				return h, nil
			}
			h.todoDialog.SetTodos(todos)
		}
	}
```

Note: After `SetTodos()`, `rebuildCols()` restores the cursor to the card's new column by ID automatically — no extra cursor manipulation needed.

**Step 4: Final build and test**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: `ok  ghe.spotify.net/mnicholson/hangar/internal/ui` (with 2 pre-existing failures still showing as FAIL lines).

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/todo_dialog_test.go
git commit -m "feat(kanban): wire move-card actions in home.go"
```

---

### Task 7: Manual smoke test

**Step 1: Build the binary**

```bash
go build -o /tmp/hangar-kanban ./cmd/hangar
```

Expected: no errors.

**Step 2: Manual verification checklist**

Run `/tmp/hangar-kanban` and open a project with todos (press `t`):

- [ ] Board shows 4 columns: todo / in progress / in review / done
- [ ] `←`/`→` moves focus between columns (focused header highlighted in cyan)
- [ ] `↑`/`↓` moves cursor within a column
- [ ] `n` opens new-todo form; saving lands card in the focused column's status
- [ ] `e` opens edit form for selected card
- [ ] `d` deletes selected card
- [ ] `s` opens status picker; changing status moves card to correct column
- [ ] `shift+→` moves card right one column; cursor follows
- [ ] `shift+←` moves card left one column; cursor follows
- [ ] `shift+→` on a `done` card: no-op
- [ ] `shift+←` on a `todo` card: no-op
- [ ] Card with linked session shows `⬡`
- [ ] `enter` on card with no session opens new-session dialog
- [ ] `enter` on card with linked session attaches to that session
- [ ] `esc`/`t` closes dialog
- [ ] Creating an orphaned todo: orphaned column appears; disappears when orphaned todo deleted
- [ ] Empty board shows "No todos yet — press n to create one"

**Step 3: Final commit if any fixes needed**

```bash
git add -p
git commit -m "fix(kanban): <describe fix>"
```
