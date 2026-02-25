# Todo List Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a per-project todo list to Hangar with automatic status transitions driven by worktree/PR lifecycle events.

**Architecture:** New `todos` table in the per-profile SQLite DB (`statedb` layer), domain `Todo` type and storage methods in the `session` package, a `TodoDialog` in the `ui` package following the existing dialog pattern, and lifecycle hooks wired into `home.go`.

**Tech Stack:** Go, SQLite (`modernc.org/sqlite`), Bubble Tea, lipgloss

---

## Task 1: Add `todos` table to statedb

**Files:**
- Modify: `internal/statedb/statedb.go`
- Modify: `internal/statedb/statedb_test.go`

### Step 1: Write failing tests for todo CRUD

Add to `internal/statedb/statedb_test.go`:

```go
func TestSaveTodo(t *testing.T) {
    db := newTestDB(t)
    row := &TodoRow{
        ID:          "todo-1",
        ProjectPath: "/projects/myapp",
        Title:       "fix auth bug",
        Description: "tokens expire too early",
        Status:      "todo",
        SessionID:   "",
        Order:       0,
        CreatedAt:   time.Unix(1000, 0),
        UpdatedAt:   time.Unix(1000, 0),
    }
    if err := db.SaveTodo(row); err != nil {
        t.Fatalf("SaveTodo: %v", err)
    }

    todos, err := db.LoadTodos("/projects/myapp")
    if err != nil {
        t.Fatalf("LoadTodos: %v", err)
    }
    if len(todos) != 1 {
        t.Fatalf("expected 1 todo, got %d", len(todos))
    }
    got := todos[0]
    if got.Title != "fix auth bug" {
        t.Errorf("Title: got %q want %q", got.Title, "fix auth bug")
    }
    if got.Status != "todo" {
        t.Errorf("Status: got %q want %q", got.Status, "todo")
    }
}

func TestLoadTodos_ProjectScoped(t *testing.T) {
    db := newTestDB(t)
    _ = db.SaveTodo(&TodoRow{ID: "a", ProjectPath: "/proj/alpha", Title: "alpha", Status: "todo", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)})
    _ = db.SaveTodo(&TodoRow{ID: "b", ProjectPath: "/proj/beta", Title: "beta", Status: "todo", CreatedAt: time.Unix(2, 0), UpdatedAt: time.Unix(2, 0)})

    todos, _ := db.LoadTodos("/proj/alpha")
    if len(todos) != 1 || todos[0].ID != "a" {
        t.Errorf("expected only alpha's todo, got %v", todos)
    }
}

func TestUpdateTodoStatus(t *testing.T) {
    db := newTestDB(t)
    _ = db.SaveTodo(&TodoRow{ID: "t1", ProjectPath: "/p", Title: "task", Status: "todo", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)})
    if err := db.UpdateTodoStatus("t1", "in_progress", "sess-123"); err != nil {
        t.Fatalf("UpdateTodoStatus: %v", err)
    }
    todos, _ := db.LoadTodos("/p")
    if todos[0].Status != "in_progress" {
        t.Errorf("Status: got %q want in_progress", todos[0].Status)
    }
    if todos[0].SessionID != "sess-123" {
        t.Errorf("SessionID: got %q want sess-123", todos[0].SessionID)
    }
}

func TestDeleteTodo(t *testing.T) {
    db := newTestDB(t)
    _ = db.SaveTodo(&TodoRow{ID: "del", ProjectPath: "/p", Title: "gone", Status: "todo", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)})
    if err := db.DeleteTodo("del"); err != nil {
        t.Fatalf("DeleteTodo: %v", err)
    }
    todos, _ := db.LoadTodos("/p")
    if len(todos) != 0 {
        t.Errorf("expected 0 todos after delete, got %d", len(todos))
    }
}

func TestFindTodoBySessionID(t *testing.T) {
    db := newTestDB(t)
    _ = db.SaveTodo(&TodoRow{ID: "s1", ProjectPath: "/p", Title: "linked", Status: "in_progress", SessionID: "sess-abc", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)})
    _ = db.SaveTodo(&TodoRow{ID: "s2", ProjectPath: "/p", Title: "unlinked", Status: "todo", CreatedAt: time.Unix(2, 0), UpdatedAt: time.Unix(2, 0)})

    row, err := db.FindTodoBySessionID("sess-abc")
    if err != nil {
        t.Fatalf("FindTodoBySessionID: %v", err)
    }
    if row == nil || row.ID != "s1" {
        t.Errorf("expected s1, got %v", row)
    }

    missing, _ := db.FindTodoBySessionID("no-such")
    if missing != nil {
        t.Errorf("expected nil for unknown session, got %v", missing)
    }
}
```

### Step 2: Run to confirm tests fail

```
go test ./internal/statedb/... -run "TestSaveTodo|TestLoadTodos|TestUpdateTodo|TestDeleteTodo|TestFindTodo" -v
```
Expected: FAIL — `TodoRow` undefined, `SaveTodo` undefined.

### Step 3: Add `TodoRow` struct and `todos` table to statedb.go

**In `internal/statedb/statedb.go`:**

1. Bump `SchemaVersion`:
```go
const SchemaVersion = 2
```

2. Add `TodoRow` struct after `GroupRow`:
```go
// TodoRow represents a todo item row in the database.
type TodoRow struct {
    ID          string
    ProjectPath string
    Title       string
    Description string
    Status      string // todo | in_progress | in_review | done | orphaned
    SessionID   string // soft FK to instances.id (empty = unlinked)
    Order       int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

3. Add `todos` table creation inside `Migrate()`, after the `instance_heartbeats` block:
```go
// todos table
if _, err := tx.Exec(`
    CREATE TABLE IF NOT EXISTS todos (
        id           TEXT PRIMARY KEY,
        project_path TEXT NOT NULL,
        title        TEXT NOT NULL,
        description  TEXT NOT NULL DEFAULT '',
        status       TEXT NOT NULL DEFAULT 'todo',
        session_id   TEXT NOT NULL DEFAULT '',
        sort_order   INTEGER NOT NULL DEFAULT 0,
        created_at   INTEGER NOT NULL,
        updated_at   INTEGER NOT NULL
    )
`); err != nil {
    return fmt.Errorf("statedb: create todos: %w", err)
}
```

4. Add CRUD methods at the end of the file (after the `--- Metadata ---` section):

```go
// --- Todo CRUD ---

// SaveTodo inserts or replaces a single todo row.
func (s *StateDB) SaveTodo(row *TodoRow) error {
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO todos
            (id, project_path, title, description, status, session_id, sort_order, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        row.ID, row.ProjectPath, row.Title, row.Description,
        row.Status, row.SessionID, row.Order,
        row.CreatedAt.Unix(), row.UpdatedAt.Unix(),
    )
    return err
}

// LoadTodos returns all todos for a given project path, ordered by sort_order.
func (s *StateDB) LoadTodos(projectPath string) ([]*TodoRow, error) {
    rows, err := s.db.Query(`
        SELECT id, project_path, title, description, status, session_id, sort_order, created_at, updated_at
        FROM todos WHERE project_path = ? ORDER BY sort_order, created_at
    `, projectPath)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []*TodoRow
    for rows.Next() {
        r := &TodoRow{}
        var createdUnix, updatedUnix int64
        if err := rows.Scan(
            &r.ID, &r.ProjectPath, &r.Title, &r.Description,
            &r.Status, &r.SessionID, &r.Order,
            &createdUnix, &updatedUnix,
        ); err != nil {
            return nil, err
        }
        r.CreatedAt = time.Unix(createdUnix, 0)
        r.UpdatedAt = time.Unix(updatedUnix, 0)
        result = append(result, r)
    }
    return result, rows.Err()
}

// DeleteTodo removes a todo by ID.
func (s *StateDB) DeleteTodo(id string) error {
    _, err := s.db.Exec("DELETE FROM todos WHERE id = ?", id)
    return err
}

// UpdateTodoStatus updates the status and session_id for a todo.
func (s *StateDB) UpdateTodoStatus(id, status, sessionID string) error {
    _, err := s.db.Exec(
        "UPDATE todos SET status = ?, session_id = ?, updated_at = ? WHERE id = ?",
        status, sessionID, time.Now().Unix(), id,
    )
    return err
}

// FindTodoBySessionID returns the todo linked to the given session ID, or nil if none.
func (s *StateDB) FindTodoBySessionID(sessionID string) (*TodoRow, error) {
    if sessionID == "" {
        return nil, nil
    }
    r := &TodoRow{}
    var createdUnix, updatedUnix int64
    err := s.db.QueryRow(`
        SELECT id, project_path, title, description, status, session_id, sort_order, created_at, updated_at
        FROM todos WHERE session_id = ? LIMIT 1
    `, sessionID).Scan(
        &r.ID, &r.ProjectPath, &r.Title, &r.Description,
        &r.Status, &r.SessionID, &r.Order,
        &createdUnix, &updatedUnix,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    r.CreatedAt = time.Unix(createdUnix, 0)
    r.UpdatedAt = time.Unix(updatedUnix, 0)
    return r, nil
}
```

### Step 4: Run tests to confirm they pass

```
go test ./internal/statedb/... -v
```
Expected: ALL PASS (including pre-existing tests).

### Step 5: Commit

```bash
git add internal/statedb/statedb.go internal/statedb/statedb_test.go
git commit -m "feat(statedb): add todos table with CRUD methods"
```

---

## Task 2: Todo domain model and Storage methods

**Files:**
- Create: `internal/session/todo.go`

### Step 1: Write the file

```go
package session

import (
    "fmt"
    "time"

    "ghe.spotify.net/mnicholson/hangar/internal/statedb"
)

// TodoStatus represents the lifecycle state of a todo item.
type TodoStatus string

const (
    TodoStatusTodo       TodoStatus = "todo"
    TodoStatusInProgress TodoStatus = "in_progress"
    TodoStatusInReview   TodoStatus = "in_review"
    TodoStatusDone       TodoStatus = "done"
    TodoStatusOrphaned   TodoStatus = "orphaned"
)

// Todo represents a work item tied to a project.
type Todo struct {
    ID          string
    ProjectPath string
    Title       string
    Description string
    Status      TodoStatus
    SessionID   string // empty = unlinked
    Order       int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// generateTodoID generates a unique todo ID using the same pattern as session IDs.
func generateTodoID() string {
    return fmt.Sprintf("todo-%s-%d", randomString(8), time.Now().Unix())
}

// NewTodo creates a new Todo for the given project with todo status.
func NewTodo(title, description, projectPath string) *Todo {
    now := time.Now()
    return &Todo{
        ID:          generateTodoID(),
        ProjectPath: projectPath,
        Title:       title,
        Description: description,
        Status:      TodoStatusTodo,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
}

func todoFromRow(r *statedb.TodoRow) *Todo {
    return &Todo{
        ID:          r.ID,
        ProjectPath: r.ProjectPath,
        Title:       r.Title,
        Description: r.Description,
        Status:      TodoStatus(r.Status),
        SessionID:   r.SessionID,
        Order:       r.Order,
        CreatedAt:   r.CreatedAt,
        UpdatedAt:   r.UpdatedAt,
    }
}

func todoToRow(t *Todo) *statedb.TodoRow {
    return &statedb.TodoRow{
        ID:          t.ID,
        ProjectPath: t.ProjectPath,
        Title:       t.Title,
        Description: t.Description,
        Status:      string(t.Status),
        SessionID:   t.SessionID,
        Order:       t.Order,
        CreatedAt:   t.CreatedAt,
        UpdatedAt:   t.UpdatedAt,
    }
}

// --- Storage methods for todos ---

// LoadTodos returns all todos for the given project path.
func (s *Storage) LoadTodos(projectPath string) ([]*Todo, error) {
    rows, err := s.db.LoadTodos(projectPath)
    if err != nil {
        return nil, err
    }
    todos := make([]*Todo, len(rows))
    for i, r := range rows {
        todos[i] = todoFromRow(r)
    }
    return todos, nil
}

// SaveTodo inserts or updates a todo.
func (s *Storage) SaveTodo(todo *Todo) error {
    todo.UpdatedAt = time.Now()
    return s.db.SaveTodo(todoToRow(todo))
}

// DeleteTodo removes a todo by ID.
func (s *Storage) DeleteTodo(id string) error {
    return s.db.DeleteTodo(id)
}

// UpdateTodoStatus updates a todo's status and linked session ID.
func (s *Storage) UpdateTodoStatus(id string, status TodoStatus, sessionID string) error {
    return s.db.UpdateTodoStatus(id, string(status), sessionID)
}

// OrphanTodosForSession sets status to "orphaned" for any todo linked to the given session.
func (s *Storage) OrphanTodosForSession(sessionID string) error {
    row, err := s.db.FindTodoBySessionID(sessionID)
    if err != nil || row == nil {
        return err
    }
    return s.db.UpdateTodoStatus(row.ID, "orphaned", sessionID)
}

// DeleteTodosForSession removes any todo linked to the given session.
func (s *Storage) DeleteTodosForSession(sessionID string) error {
    row, err := s.db.FindTodoBySessionID(sessionID)
    if err != nil || row == nil {
        return err
    }
    return s.db.DeleteTodo(row.ID)
}

// FindTodoBySessionID returns the todo linked to the given session, or nil.
func (s *Storage) FindTodoBySessionID(sessionID string) (*Todo, error) {
    row, err := s.db.FindTodoBySessionID(sessionID)
    if err != nil || row == nil {
        return nil, err
    }
    return todoFromRow(row), nil
}
```

### Step 2: Build to confirm no compile errors

```
go build ./internal/session/...
```
Expected: success (no output).

### Step 3: Commit

```bash
git add internal/session/todo.go
git commit -m "feat(session): add Todo domain model and Storage methods"
```

---

## Task 3: TodoDialog — list view and new-todo form

**Files:**
- Create: `internal/ui/todo_dialog.go`

### Step 1: Write the dialog

```go
package ui

import (
    "fmt"
    "strings"
    "time"

    "ghe.spotify.net/mnicholson/hangar/internal/git"
    "ghe.spotify.net/mnicholson/hangar/internal/session"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// todoDialogMode controls which sub-view is active.
type todoDialogMode int

const (
    todoModeList   todoDialogMode = iota
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
    todos       []*session.Todo
    cursor      int
    mode        todoDialogMode
    errorMsg    string

    // new/edit form fields
    titleInput textinput.Model
    descInput  textinput.Model
    formFocus  int  // 0=title, 1=desc
    editingID  string // non-empty when editing

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
        titleInput:    titleInput,
        descInput:     descInput,
        statusOptions: []session.TodoStatus{
            session.TodoStatusTodo,
            session.TodoStatusInProgress,
            session.TodoStatusInReview,
            session.TodoStatusDone,
            session.TodoStatusOrphaned,
        },
    }
}

// IsVisible returns true if the dialog is open.
func (d *TodoDialog) IsVisible() bool { return d.visible }

// Show opens the dialog for the given project path with the given todos.
func (d *TodoDialog) Show(projectPath string, todos []*session.Todo) {
    d.visible = true
    d.projectPath = projectPath
    d.todos = todos
    d.cursor = 0
    d.mode = todoModeList
    d.errorMsg = ""
}

// Hide closes the dialog.
func (d *TodoDialog) Hide() { d.visible = false }

// SetSize updates dimensions.
func (d *TodoDialog) SetSize(w, h int) {
    d.width = w
    d.height = h
    d.titleInput.Width = w/2 - 6
    d.descInput.Width = w/2 - 6
}

// SetTodos replaces the current todo list (used after reloads).
func (d *TodoDialog) SetTodos(todos []*session.Todo) {
    d.todos = todos
    if d.cursor >= len(d.todos) && len(d.todos) > 0 {
        d.cursor = len(d.todos) - 1
    }
}

// SelectedTodo returns the currently selected todo, or nil.
func (d *TodoDialog) SelectedTodo() *session.Todo {
    if len(d.todos) == 0 || d.cursor >= len(d.todos) {
        return nil
    }
    return d.todos[d.cursor]
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
)

// GetFormValues returns the title, description, and editing ID after TodoActionSaveTodo.
func (d *TodoDialog) GetFormValues() (title, description, editingID string) {
    return strings.TrimSpace(d.titleInput.Value()),
        strings.TrimSpace(d.descInput.Value()),
        d.editingID
}

// GetPickedStatus returns the status chosen in the status picker.
func (d *TodoDialog) GetPickedStatus() session.TodoStatus {
    if d.statusCursor < len(d.statusOptions) {
        return d.statusOptions[d.statusCursor]
    }
    return session.TodoStatusTodo
}

// HandleKey processes a keypress and returns the action the caller should take.
func (d *TodoDialog) HandleKey(key string) TodoAction {
    switch d.mode {
    case todoModeList:
        return d.handleListKey(key)
    case todoModeNew, todoModeEdit:
        return d.handleFormKey(key)
    case todoModeStatus:
        return d.handleStatusKey(key)
    }
    return TodoActionNone
}

func (d *TodoDialog) handleListKey(key string) TodoAction {
    switch key {
    case "up", "k":
        if d.cursor > 0 {
            d.cursor--
        }
    case "down", "j":
        if d.cursor < len(d.todos)-1 {
            d.cursor++
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
        if t.Status == session.TodoStatusTodo {
            return TodoActionCreateSession
        }
        // For in_progress/in_review: attach to session (handled by caller using t.SessionID)
        if t.SessionID != "" {
            return TodoActionCreateSession // caller checks status to distinguish
        }
    case "esc", "t":
        return TodoActionClose
    }
    return TodoActionNone
}

func (d *TodoDialog) handleFormKey(key string) TodoAction {
    switch key {
    case "tab":
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
        d.mode = todoModeList
        d.errorMsg = ""
    default:
        if d.formFocus == 0 {
            d.titleInput, _ = d.titleInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
        } else {
            d.descInput, _ = d.descInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
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
        d.mode = todoModeList
        return TodoActionUpdateStatus
    case "esc":
        d.mode = todoModeList
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

// ResetFormToList returns dialog to list mode (call after successful save/update).
func (d *TodoDialog) ResetFormToList() {
    d.mode = todoModeList
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
        return d.viewList()
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
    for i, t := range d.todos {
        statusSt := todoStatusStyle(t.Status)
        icon := statusSt.Render(todoStatusIcon(t.Status))
        label := statusSt.Render(todoStatusLabel(t.Status))

        titleCol := t.Title
        if len(titleCol) > innerWidth-15 {
            titleCol = titleCol[:innerWidth-18] + "..."
        }

        gap := innerWidth - len(titleCol) - len(todoStatusLabel(t.Status)) - 4
        if gap < 1 {
            gap = 1
        }
        line := fmt.Sprintf(" %s %s%s%s", icon, titleCol, strings.Repeat(" ", gap), label)

        if i == d.cursor {
            line = lipgloss.NewStyle().
                Background(lipgloss.Color("#2a3a4a")).
                Foreground(lipgloss.Color("#ffffff")).
                Render(line)
        }
        rows = append(rows, line)

        // Show session hint for linked todos
        if t.SessionID != "" {
            hint := fmt.Sprintf("   └─ session linked")
            rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render(hint))
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
// Delegates to git.SanitizeBranchName and lowercases.
func TodoBranchName(title string) string {
    lower := strings.ToLower(title)
    return git.SanitizeBranchName(lower)
}

// Ensure time import used
var _ = time.Now
```

### Step 2: Build to confirm no compile errors

```
go build ./internal/ui/...
```
Expected: success. If `git` import causes issues, ensure `ghe.spotify.net/mnicholson/hangar/internal/git` is in the import.

### Step 3: Commit

```bash
git add internal/ui/todo_dialog.go
git commit -m "feat(ui): add TodoDialog for per-project todo list"
```

---

## Task 4: Wire TodoDialog into home.go (7 locations)

**Files:**
- Modify: `internal/ui/home.go`

### Step 1: Add struct field

Find the `Home` struct (around line 115). After `worktreeFinishDialog *WorktreeFinishDialog`, add:

```go
todoDialog     *TodoDialog
pendingTodoID  string // todo ID waiting for a session to be created from it
```

### Step 2: Initialize in NewHome()

Find the `NewHome()` function (around line 510). After `worktreeFinishDialog: NewWorktreeFinishDialog(),`, add:

```go
todoDialog: NewTodoDialog(),
```

### Step 3: Add key routing guard

Find the large condition that checks if any dialog is visible before processing keys (search for `h.worktreeFinishDialog.IsVisible()`). In `handleKeyMsg`, find the block like:
```go
if h.worktreeFinishDialog.IsVisible() {
    return h.handleWorktreeFinishDialogKey(msg)
}
```

Add before or after this block:
```go
if h.todoDialog.IsVisible() {
    return h.handleTodoDialogKey(msg)
}
```

### Step 4: Add mouse guard

Find the mouse event handler or any condition that checks multiple dialogs to block mouse events (search for `IsVisible` calls in `case tea.MouseMsg`). Add `h.todoDialog.IsVisible()` to the conditions there.

### Step 5: Add trigger key `t`

Find the main key dispatch (in `Update`, where bare keys like `w`, `n`, `d` are handled in the session list). Add:
```go
case "t":
    if !h.isAnyDialogVisible() {
        return h, h.showTodoDialog()
    }
```

If no `isAnyDialogVisible()` helper exists, guard with the same nil/invisible checks used by other keys.

### Step 6: Add SetSize call

Find `SetSize` (around line 5368). After `h.worktreeFinishDialog.SetSize(h.width, h.height)`, add:
```go
h.todoDialog.SetSize(h.width, h.height)
```

### Step 7: Add View() call

Find the `View()` method's dialog rendering section (around line 5450, where `h.worktreeFinishDialog.IsVisible()` is checked). Add:
```go
if h.todoDialog.IsVisible() {
    return h.todoDialog.View()
}
```

### Step 8: Add helper methods

At the end of `home.go`, add:

```go
// showTodoDialog loads todos for the current project and opens the dialog.
func (h *Home) showTodoDialog() tea.Cmd {
    item := h.selectedItem()
    if item == nil {
        return nil
    }
    var projectPath string
    if item.Session != nil {
        projectPath = item.Session.ProjectRepoRoot()
    } else if item.Group != nil && item.Group.DefaultPath != "" {
        projectPath = item.Group.DefaultPath
    }
    if projectPath == "" {
        return nil
    }
    todos, err := h.storage.LoadTodos(projectPath)
    if err != nil {
        h.setError(fmt.Errorf("failed to load todos: %w", err))
        return nil
    }
    h.todoDialog.Show(projectPath, todos)
    return nil
}

// handleTodoDialogKey processes key events for the todo dialog.
func (h *Home) handleTodoDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    action := h.todoDialog.HandleKey(msg.String())
    switch action {
    case TodoActionClose:
        h.todoDialog.Hide()

    case TodoActionSaveTodo:
        title, desc, editingID := h.todoDialog.GetFormValues()
        projectPath := h.todoDialog.projectPath
        if editingID == "" {
            // New todo
            todo := session.NewTodo(title, desc, projectPath)
            if err := h.storage.SaveTodo(todo); err != nil {
                h.setError(fmt.Errorf("save todo: %w", err))
                return h, nil
            }
        } else {
            // Edit existing todo — reload, update fields, save
            todos, _ := h.storage.LoadTodos(projectPath)
            for _, t := range todos {
                if t.ID == editingID {
                    t.Title = title
                    t.Description = desc
                    if err := h.storage.SaveTodo(t); err != nil {
                        h.setError(fmt.Errorf("update todo: %w", err))
                        return h, nil
                    }
                    break
                }
            }
        }
        // Reload and refresh
        todos, _ := h.storage.LoadTodos(projectPath)
        h.todoDialog.SetTodos(todos)
        h.todoDialog.ResetFormToList()

    case TodoActionDeleteTodo:
        todo := h.todoDialog.SelectedTodo()
        if todo != nil {
            if err := h.storage.DeleteTodo(todo.ID); err != nil {
                h.setError(fmt.Errorf("delete todo: %w", err))
                return h, nil
            }
            todos, _ := h.storage.LoadTodos(h.todoDialog.projectPath)
            h.todoDialog.SetTodos(todos)
        }

    case TodoActionUpdateStatus:
        todo := h.todoDialog.SelectedTodo()
        if todo != nil {
            newStatus := h.todoDialog.GetPickedStatus()
            if err := h.storage.UpdateTodoStatus(todo.ID, newStatus, todo.SessionID); err != nil {
                h.setError(fmt.Errorf("update status: %w", err))
                return h, nil
            }
            todos, _ := h.storage.LoadTodos(h.todoDialog.projectPath)
            h.todoDialog.SetTodos(todos)
        }

    case TodoActionCreateSession:
        todo := h.todoDialog.SelectedTodo()
        if todo == nil {
            return h, nil
        }
        if todo.Status != session.TodoStatusTodo || todo.SessionID != "" {
            // Already has a session — attach to it
            h.todoDialog.Hide()
            h.instancesMu.RLock()
            inst := h.instanceByID[todo.SessionID]
            h.instancesMu.RUnlock()
            if inst != nil {
                return h, h.attachSession(inst)
            }
            return h, nil
        }
        // Open new-session dialog pre-filled for this todo
        h.pendingTodoID = todo.ID
        h.todoDialog.Hide()
        branchName := TodoBranchName(todo.Title)
        h.newDialog.ShowWithWorktree(branchName, h.todoDialog.projectPath)
        return h, nil
    }
    return h, nil
}
```

**Note on `ProjectRepoRoot()`:** Check if `Instance` has this method; if not, use `inst.WorktreeRepoRoot` if set, otherwise `inst.ProjectPath`. You may need:
```go
func (inst *Instance) ProjectRepoRoot() string {
    if inst.WorktreeRepoRoot != "" {
        return inst.WorktreeRepoRoot
    }
    return inst.ProjectPath
}
```
Add this to `internal/session/instance.go` if it doesn't exist.

**Note on `newDialog.ShowWithWorktree()`:** Check if `NewDialog` has a pre-fill method. If not, use the dialog's existing `Show()` / `Reset()` and manually set the branch field. Look at how the worktree creation dialog is opened from existing code and follow the same pattern.

### Step 9: Build

```
go build ./...
```
Expected: success. Fix any compile errors.

### Step 10: Commit

```bash
git add internal/ui/home.go internal/session/instance.go
git commit -m "feat(ui): wire TodoDialog into home.go"
```

---

## Task 5: Lifecycle hooks — orphan, worktree finish, PR transitions

**Files:**
- Modify: `internal/ui/home.go`

### Step 1: Orphan todos on session delete

In the `sessionDeletedMsg` handler (around line 2362), after `h.storage.DeleteInstance(msg.deletedID)`, add:

```go
// Orphan any todo linked to this session
if err := h.storage.OrphanTodosForSession(msg.deletedID); err != nil {
    uiLog.Warn("orphan_todo_err", slog.String("id", msg.deletedID), slog.String("err", err.Error()))
}
// Refresh todo dialog if visible
if h.todoDialog.IsVisible() {
    todos, _ := h.storage.LoadTodos(h.todoDialog.projectPath)
    h.todoDialog.SetTodos(todos)
}
```

### Step 2: Delete todos on worktree finish

In the `worktreeFinishResultMsg` handler (around line 2856), after `h.storage.DeleteInstance(msg.sessionID)`, add:

```go
// Remove any todo linked to this worktree session
if err := h.storage.DeleteTodosForSession(msg.sessionID); err != nil {
    uiLog.Warn("delete_todo_err", slog.String("id", msg.sessionID), slog.String("err", err.Error()))
}
```

### Step 3: Update todo status on PR events

In the `prFetchedMsg` handler (around line 2794), after updating the PR cache, add:

```go
// Update linked todo status based on PR state
if msg.pr != nil && msg.pr.State != "" {
    var newStatus session.TodoStatus
    switch msg.pr.State {
    case "OPEN", "DRAFT":
        newStatus = session.TodoStatusInReview
    case "MERGED":
        newStatus = session.TodoStatusDone
    }
    if newStatus != "" {
        if err := h.storage.UpdateTodoStatus("", newStatus, msg.sessionID); err == nil {
            // noop: UpdateTodoStatus needs the ID not sessionID — use FindTodoBySessionID
        }
        if todo, err := h.storage.FindTodoBySessionID(msg.sessionID); err == nil && todo != nil {
            // Only auto-advance; don't regress (e.g., don't go done→in_review)
            shouldUpdate := false
            switch newStatus {
            case session.TodoStatusInReview:
                shouldUpdate = todo.Status == session.TodoStatusInProgress
            case session.TodoStatusDone:
                shouldUpdate = todo.Status == session.TodoStatusInProgress || todo.Status == session.TodoStatusInReview
            }
            if shouldUpdate {
                _ = h.storage.UpdateTodoStatus(todo.ID, newStatus, todo.SessionID)
            }
        }
        // Refresh todo dialog if open
        if h.todoDialog.IsVisible() {
            todos, _ := h.storage.LoadTodos(h.todoDialog.projectPath)
            h.todoDialog.SetTodos(todos)
        }
    }
}
```

### Step 4: Link todo when session created from one

Find where new sessions are saved after the `NewDialog` submit (around line 2233, where `CRITICAL: Save the new session` comment is). After the new session is saved, add:

```go
// Link pending todo to new session
if h.pendingTodoID != "" {
    if err := h.storage.UpdateTodoStatus(h.pendingTodoID, session.TodoStatusInProgress, newInst.ID); err != nil {
        uiLog.Warn("link_todo_err", slog.String("todo_id", h.pendingTodoID), slog.String("err", err.Error()))
    }
    h.pendingTodoID = ""
}
```

### Step 5: Build and run tests

```
go build ./...
go test ./internal/...
```
Expected: all tests pass (2 pre-existing TUI failures are OK).

### Step 6: Commit

```bash
git add internal/ui/home.go
git commit -m "feat(ui): add todo lifecycle hooks (orphan, finish, PR transitions)"
```

---

## Task 6: Manual testing and final polish

### Step 1: Build and run

```
go run ./cmd/hangar
```

Exercise the following flows manually:
1. Open Hangar, navigate to a session in a project, press `t` — todos view should open
2. Press `n`, enter a title and description, press `enter` — todo appears in list
3. Press `s` — status picker appears; change status, press `enter` — status updates
4. Press `e` — edit form opens with existing values
5. Press `enter` on a `todo`-status item — new-session dialog opens pre-filled
6. After creating a session, re-open todos — item shows `in_progress`
7. Press `d` — todo deleted with confirmation
8. Delete a session that has a linked todo — re-open todos, item shows `orphaned`
9. Use worktree finish on a session with a linked todo — re-open todos, item should be gone

### Step 2: Fix any issues found

### Step 3: Final commit

```bash
git add -A
git commit -m "feat: todo list — manual testing fixes"
```

---

## Summary

| Task | Files | Tests |
|------|-------|-------|
| 1. statedb todos table | `internal/statedb/statedb.go` | `statedb_test.go` |
| 2. Todo domain model | `internal/session/todo.go` | (build test) |
| 3. TodoDialog | `internal/ui/todo_dialog.go` | (build test) |
| 4. Wire into home.go | `internal/ui/home.go` | (build test) |
| 5. Lifecycle hooks | `internal/ui/home.go` | (build test) |
| 6. Manual testing | — | manual |
