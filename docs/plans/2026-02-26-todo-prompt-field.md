# Todo Prompt Field Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an optional `Prompt` field to todos that is automatically sent to the session when one is created from the todo.

**Architecture:** Propagate a `Prompt string` field through the data layer (statedb → session.Todo), expose it in the form UI (third tab-stop in todo dialog), store a `pendingTodoPrompt` on the Home model, and deliver it via `inst.SendText` after a 4-second delay (reusing the established review-session pattern).

**Tech Stack:** Go, Bubble Tea (charmbracelet), SQLite (modernc.org/sqlite), lipgloss

---

### Task 1: Add `Prompt` to `TodoRow` and migrate the database schema

**Files:**
- Modify: `internal/statedb/statedb.go`

**Step 1: Add `Prompt` to `TodoRow` struct**

In `statedb.go`, locate `TodoRow` (around line 59) and add `Prompt` after `Description`:

```go
type TodoRow struct {
	ID          string
	ProjectPath string
	Title       string
	Description string
	Prompt      string // optional; sent to session on creation
	Status      string
	SessionID   string
	Order       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

**Step 2: Bump SchemaVersion and add the ALTER TABLE migration**

Change `SchemaVersion = 2` to `SchemaVersion = 3`.

Inside `Migrate()`, directly after the `CREATE TABLE IF NOT EXISTS todos (...)` block (around line 228), add:

```go
// Migration v3: add prompt column to todos table for existing databases.
// ALTER TABLE ADD COLUMN is idempotent in SQLite when we ignore "duplicate column" errors.
if _, err := tx.Exec(`ALTER TABLE todos ADD COLUMN prompt TEXT NOT NULL DEFAULT ''`); err != nil {
	if !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("statedb: add prompt column: %w", err)
	}
}
```

Make sure `"strings"` is already in the imports (it is — check the top of the file).

**Step 3: Update `SaveTodo` to include `prompt`**

Replace the `SaveTodo` SQL:

```go
func (s *StateDB) SaveTodo(row *TodoRow) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO todos
			(id, project_path, title, description, prompt, status, session_id, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		row.ID, row.ProjectPath, row.Title, row.Description, row.Prompt,
		row.Status, row.SessionID, row.Order,
		row.CreatedAt.Unix(), row.UpdatedAt.Unix(),
	)
	return err
}
```

**Step 4: Update `LoadTodos` to scan `prompt`**

Replace the SELECT and Scan in `LoadTodos`:

```go
func (s *StateDB) LoadTodos(projectPath string) ([]*TodoRow, error) {
	rows, err := s.db.Query(`
		SELECT id, project_path, title, description, prompt, status, session_id, sort_order, created_at, updated_at
		FROM todos WHERE project_path = ? ORDER BY sort_order, created_at
	`, projectPath)
	// ... (rows.Next loop unchanged except Scan)
	if err := rows.Scan(
		&r.ID, &r.ProjectPath, &r.Title, &r.Description, &r.Prompt,
		&r.Status, &r.SessionID, &r.Order,
		&createdUnix, &updatedUnix,
	); err != nil {
		return nil, err
	}
```

**Step 5: Update `FindTodoBySessionID` to scan `prompt`**

Replace the SELECT and Scan in `FindTodoBySessionID`:

```go
err := s.db.QueryRow(`
	SELECT id, project_path, title, description, prompt, status, session_id, sort_order, created_at, updated_at
	FROM todos WHERE session_id = ? LIMIT 1
`, sessionID).Scan(
	&r.ID, &r.ProjectPath, &r.Title, &r.Description, &r.Prompt,
	&r.Status, &r.SessionID, &r.Order,
	&createdUnix, &updatedUnix,
)
```

**Step 6: Run the statedb tests**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/todo-prompt
go test ./internal/statedb/... -v -run TestSave
```

Expected: PASS. The `Prompt` field defaults to `""` in existing test rows so no existing tests break.

**Step 7: Commit**

```bash
git add internal/statedb/statedb.go
git commit -m "feat(statedb): add prompt column to todos (schema v3)"
```

---

### Task 2: Add `Prompt` to the `Todo` domain model

**Files:**
- Modify: `internal/session/todo.go`

**Step 1: Add `Prompt` to the `Todo` struct**

```go
type Todo struct {
	ID          string
	ProjectPath string
	Title       string
	Description string
	Prompt      string // optional; sent to session on creation
	Status      TodoStatus
	SessionID   string
	Order       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

**Step 2: Update `NewTodo` to accept a `prompt` parameter**

```go
func NewTodo(title, description, prompt, projectPath string) *Todo {
	now := time.Now()
	return &Todo{
		ID:          generateTodoID(),
		ProjectPath: projectPath,
		Title:       title,
		Description: description,
		Prompt:      prompt,
		Status:      TodoStatusTodo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
```

**Step 3: Update `todoFromRow` and `todoToRow`**

In `todoFromRow`:
```go
func todoFromRow(r *statedb.TodoRow) *Todo {
	return &Todo{
		ID:          r.ID,
		ProjectPath: r.ProjectPath,
		Title:       r.Title,
		Description: r.Description,
		Prompt:      r.Prompt,
		Status:      TodoStatus(r.Status),
		SessionID:   r.SessionID,
		Order:       r.Order,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
```

In `todoToRow`:
```go
func todoToRow(t *Todo) *statedb.TodoRow {
	return &statedb.TodoRow{
		ID:          t.ID,
		ProjectPath: t.ProjectPath,
		Title:       t.Title,
		Description: t.Description,
		Prompt:      t.Prompt,
		Status:      string(t.Status),
		SessionID:   t.SessionID,
		Order:       t.Order,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
```

**Step 4: Build to catch compile errors**

```bash
go build ./internal/session/... ./internal/statedb/...
```

Expected: compile error in `home.go` because `NewTodo` now requires 4 args — that's expected, fix in Task 4.

**Step 5: Commit (session package only — home.go fix is next)**

```bash
git add internal/session/todo.go
git commit -m "feat(session): add Prompt field to Todo"
```

---

### Task 3: Update `TodoDialog` form with a third `Prompt` input

**Files:**
- Modify: `internal/ui/todo_dialog.go`
- Modify: `internal/ui/todo_dialog_test.go`

**Step 1: Add `promptInput` field to `TodoDialog`**

In the `TodoDialog` struct, after `descInput`:
```go
promptInput   textinput.Model
formFocus     int  // 0=title, 1=desc, 2=prompt
```

**Step 2: Initialize `promptInput` in `NewTodoDialog`**

After the `descInput` initialization:
```go
promptInput := textinput.New()
promptInput.Placeholder = "Prompt (optional, sent when session starts)"
promptInput.CharLimit = 2000
promptInput.Width = 50
```

And include it in the returned struct:
```go
return &TodoDialog{
	titleInput:  titleInput,
	descInput:   descInput,
	promptInput: promptInput,
	// ... rest unchanged
}
```

**Step 3: Update `SetSize` to set `promptInput.Width`**

```go
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
```

**Step 4: Update `GetFormValues` to return `prompt`**

```go
func (d *TodoDialog) GetFormValues() (title, description, prompt, editingID string, status session.TodoStatus) {
	return strings.TrimSpace(d.titleInput.Value()),
		strings.TrimSpace(d.descInput.Value()),
		strings.TrimSpace(d.promptInput.Value()),
		d.editingID,
		d.newTodoStatus
}
```

**Step 5: Update `openNewForm` to reset `promptInput`**

```go
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
	if d.selectedCol < len(d.cols) {
		d.newTodoStatus = d.cols[d.selectedCol].status
	} else {
		d.newTodoStatus = session.TodoStatusTodo
	}
}
```

**Step 6: Update `openEditForm` to populate `promptInput`**

```go
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
	d.newTodoStatus = t.Status
}
```

**Step 7: Update `handleFormKey` to cycle through 3 fields**

Replace the existing `tab`/`shift+tab` case and the default key dispatch:

```go
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
		default:
			d.promptInput, _ = d.promptInput.Update(msg)
		}
	}
	return TodoActionNone
}
```

**Step 8: Update `viewForm` to render the prompt field**

Replace the `viewForm` method. Add prompt label/input after description:

```go
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
	promptLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("Prompt (optional):")
	switch d.formFocus {
	case 0:
		titleLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff")).Render("Title:")
	case 1:
		descLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff")).Render("Description:")
	case 2:
		promptLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff")).Render("Prompt (optional):")
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
```

**Step 9: Update `renderDetailPanel` to show a prompt indicator**

After the description section, add prompt display. Replace `renderDetailPanel`:

```go
func (d *TodoDialog) renderDetailPanel(innerW int) string {
	t := d.SelectedTodo()
	if t == nil {
		return ""
	}

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("description")

	var body string
	if t.Description == "" {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("#3a4a5a")).Render("no description")
	} else {
		textWidth := innerW - 4
		if textWidth < 10 {
			textWidth = 10
		}
		wrapped := wordWrapText(t.Description, textWidth, 3)
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0ccd8")).Render(wrapped)
	}

	content := label + "\n" + body

	if t.Prompt != "" {
		promptLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#5a6a7a")).Render("⌨ prompt")
		textWidth := innerW - 4
		if textWidth < 10 {
			textWidth = 10
		}
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
```

**Step 10: Fix the failing test in `todo_dialog_test.go`**

Find the one test that calls `GetFormValues` with 4 return values (line ~225):

```go
// Before:
_, _, _, status := d.GetFormValues()
// After:
_, _, _, _, status := d.GetFormValues()
```

**Step 11: Run todo dialog tests**

```bash
go test ./internal/ui/... -v -run TestTodoDialog
```

Expected: PASS (pre-existing failures for `TestNewDialog_*` are unrelated — don't fix those).

**Step 12: Commit**

```bash
git add internal/ui/todo_dialog.go internal/ui/todo_dialog_test.go
git commit -m "feat(ui): add prompt field to todo form (3-field tab cycle)"
```

---

### Task 4: Wire prompt into `home.go`

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add `pendingTodoPrompt` field and `todoPromptSentMsg` type**

In the `Home` struct (near `pendingTodoID`, around line 148), add:
```go
pendingTodoPrompt string // prompt to send when the pending todo's session starts
```

Near the top of `home.go` where message types are defined (look for `reviewPromptSentMsg` around line 542), add:
```go
// todoPromptSentMsg is returned after the todo's initial prompt is delivered.
type todoPromptSentMsg struct{}
```

**Step 2: Handle `todoPromptSentMsg`**

Find `case reviewPromptSentMsg:` (around line 3143) and add below it:
```go
case todoPromptSentMsg:
	return h, nil
```

**Step 3: Capture `todo.Prompt` in `TodoActionCreateSession`**

Find the `TodoActionCreateSession` case (around line 10247). After `h.pendingTodoID = todo.ID`, add:
```go
h.pendingTodoPrompt = todo.Prompt
```

So the block becomes:
```go
branchName := TodoBranchName(todo.Title)
h.pendingTodoID = todo.ID
h.pendingTodoPrompt = todo.Prompt
h.todoDialog.Hide()
h.newDialog.ShowInGroupWithWorktree(groupPath, groupName, projectPath, branchName)
return h, nil
```

**Step 4: Clear `pendingTodoPrompt` on cancel**

There are two cancel spots:

1. Esc in the new dialog (around line 3857):
```go
case "esc":
	h.newDialog.Hide()
	h.pendingTodoID = ""
	h.pendingTodoPrompt = ""
	h.clearError()
	return h, nil
```

2. ConfirmCreateDirectory "n"/"esc" (around line 5014):
```go
case "n", "N", "esc":
	h.confirmDialog.Hide()
	h.pendingTodoID = ""
	h.pendingTodoPrompt = ""
	return h, nil
```

**Step 5: Deliver the prompt in `sessionCreatedMsg` handler**

Find the block in `sessionCreatedMsg` (around line 2421) that links the todo:

```go
// Link pending todo to the newly created session
if h.pendingTodoID != "" {
	if err := h.storage.UpdateTodoStatus(h.pendingTodoID, session.TodoStatusInProgress, msg.instance.ID); err != nil {
		uiLog.Warn("link_todo_err", slog.String("todo", h.pendingTodoID), slog.String("err", err.Error()))
	}
	h.pendingTodoID = ""
}

// Start fetching preview for the new session
return h, h.fetchPreview(msg.instance)
```

Replace the last two lines with prompt delivery logic:

```go
// Send initial prompt if the todo had one
cmds := []tea.Cmd{h.fetchPreview(msg.instance)}
if h.pendingTodoPrompt != "" {
	capturedInst := msg.instance
	capturedPrompt := h.pendingTodoPrompt
	h.pendingTodoPrompt = ""
	cmds = append(cmds, func() tea.Msg {
		time.Sleep(4 * time.Second)
		if err := capturedInst.SendText(capturedPrompt); err != nil {
			uiLog.Warn("todo_prompt_send_failed",
				slog.String("id", capturedInst.ID),
				slog.String("err", err.Error()))
		}
		return todoPromptSentMsg{}
	})
}
return h, tea.Batch(cmds...)
```

**Step 6: Update `TodoActionSaveTodo` to pass prompt through**

Find the `TodoActionSaveTodo` handler (around line 10146). The call to `GetFormValues` now returns 5 values:

```go
case TodoActionSaveTodo:
	title, desc, prompt, editingID, newStatus := h.todoDialog.GetFormValues()
	projectPath := h.todoDialog.projectPath
	if editingID == "" {
		todo := session.NewTodo(title, desc, prompt, projectPath)
		todo.Status = newStatus
		if err := h.storage.SaveTodo(todo); err != nil {
			h.setError(fmt.Errorf("save todo: %w", err))
			return h, nil
		}
	} else {
		todos, err := h.storage.LoadTodos(projectPath)
		if err != nil {
			h.setError(fmt.Errorf("reload todos: %w", err))
			return h, nil
		}
		for _, t := range todos {
			if t.ID == editingID {
				t.Title = title
				t.Description = desc
				t.Prompt = prompt
				if err := h.storage.SaveTodo(t); err != nil {
					h.setError(fmt.Errorf("update todo: %w", err))
					return h, nil
				}
				break
			}
		}
	}
```

**Step 7: Build to verify no compile errors**

```bash
go build ./...
```

Expected: clean build.

**Step 8: Run all tests**

```bash
go test ./... 2>&1 | grep -v "TestNewDialog_WorktreeToggle_ViaKeyPress\|TestNewDialog_TypingResetsSuggestionNavigation"
```

Expected: all tests pass (the two pre-existing failures are excluded by the grep).

**Step 9: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): deliver todo prompt to new session on creation"
```

---

### Task 5: Verify end-to-end behavior manually

**Step 1: Build the binary**

```bash
go build -o /tmp/hangar-test ./cmd/hangar
```

**Step 2: Smoke test**

- Open hangar: `/tmp/hangar-test`
- Press `t` on any project with sessions
- Press `n` to open the new todo form
- Tab through title → description → prompt fields — verify focus highlight moves correctly
- Fill in title, optional description, and a prompt like `Please implement a hello world function`
- Press Enter to save
- Select the new todo and press Enter to start a session
- After 4 seconds, verify the prompt appears in the new session's tmux pane

**Step 3: Test edit round-trip**

- Open the todo dialog, select the todo created above, press `e`
- Verify all 3 fields are pre-populated correctly
- Edit the prompt and save
- Verify the detail panel shows the updated prompt under the `⌨ prompt` label

**Step 4: Test empty prompt (no-op)**

- Create a todo with no prompt
- Start a session from it
- Verify no text is sent to the session after 4 seconds

**Step 5: Final commit**

```bash
git add .
git commit -m "chore: verify todo prompt feature end-to-end"
```

---

## Summary of Changes

| File | Change |
|------|--------|
| `internal/statedb/statedb.go` | `TodoRow.Prompt`, `SchemaVersion=3`, ALTER TABLE migration, SQL updates |
| `internal/session/todo.go` | `Todo.Prompt`, updated `NewTodo(title, desc, prompt, projectPath)`, `todoFromRow`, `todoToRow` |
| `internal/ui/todo_dialog.go` | `promptInput` field, 3-way tab cycle, updated `GetFormValues`, `openNewForm`, `openEditForm`, `SetSize`, `viewForm`, `renderDetailPanel` |
| `internal/ui/todo_dialog_test.go` | Fix `GetFormValues` call-site (5 return values now) |
| `internal/ui/home.go` | `pendingTodoPrompt` field, `todoPromptSentMsg`, capture/clear/deliver prompt |
