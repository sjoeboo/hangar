package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sjoeboo/hangar/internal/session"
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
	if cols[0].label == "" {
		t.Error("col 0 label should not be empty")
	}
	if cols[1].label == "" {
		t.Error("col 1 label should not be empty")
	}
	if cols[2].label == "" {
		t.Error("col 2 label should not be empty")
	}
	if cols[3].label == "" {
		t.Error("col 3 label should not be empty")
	}
}

func TestBuildColumns_UnrecognisedStatusGoesToOrphaned(t *testing.T) {
	todos := []*session.Todo{
		makeTodo("x", session.TodoStatus("future_status")),
	}
	cols := buildColumns(todos)
	if len(cols) != 5 {
		t.Fatalf("expected 5 columns (orphaned), got %d", len(cols))
	}
	if len(cols[4].todos) != 1 || cols[4].todos[0].ID != "x" {
		t.Errorf("expected future_status todo in orphaned column, got %v", cols[4])
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

func TestTodoDialog_SetTodos_ClampsWhenSelectedDeleted(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusTodo),
	}
	d.Show("/proj", "", "", todos)
	// Move cursor to row 1 (todo b)
	d.selectedRow[0] = 1
	// Reload with only a — b is gone, cursor should clamp to row 0
	d.SetTodos([]*session.Todo{makeTodo("a", session.TodoStatusTodo)})
	got := d.SelectedTodo()
	if got == nil || got.ID != "a" {
		t.Errorf("expected cursor clamped to a, got %v", got)
	}
}

func TestTodoDialog_SetTodos_FollowsTodoAcrossColumns(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusInProgress),
	}
	d.Show("/proj", "", "", todos)
	// Select b in col 1
	d.selectedCol = 1
	// Reload with b now in Done (col 3)
	d.SetTodos([]*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusDone),
	})
	got := d.SelectedTodo()
	if got == nil || got.ID != "b" {
		t.Errorf("expected cursor to follow b to Done column, got %v", got)
	}
	if d.selectedCol != 3 {
		t.Errorf("expected selectedCol=3 (Done), got %d", d.selectedCol)
	}
}

func TestTodoDialog_SelectedTodo_BeforeShow(t *testing.T) {
	d := NewTodoDialog()
	// No Show() called — must not panic and must return nil
	if d.SelectedTodo() != nil {
		t.Error("expected nil SelectedTodo before Show()")
	}
}

func TestTodoDialog_HandleKanban_LeftRight(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusTodo),
		makeTodo("b", session.TodoStatusInProgress),
	}
	d.Show("/proj", "", "", todos)

	if d.selectedCol != 0 {
		t.Fatalf("expected col 0, got %d", d.selectedCol)
	}
	d.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	if d.selectedCol != 1 {
		t.Errorf("expected col 1 after right, got %d", d.selectedCol)
	}
	d.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if d.selectedCol != 0 {
		t.Errorf("expected col 0 after left, got %d", d.selectedCol)
	}
	d.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if d.selectedCol != 0 {
		t.Errorf("expected col 0 (no wrap at left boundary), got %d", d.selectedCol)
	}
}

func TestTodoDialog_HandleKanban_NewPreSelectsColumnStatus(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	d.selectedCol = 1
	d.HandleKey(keyMsg("n"))
	_, _, _, _, status := d.GetFormValues()
	if status != session.TodoStatusInProgress {
		t.Errorf("expected InProgress status for new todo in col 1, got %s", status)
	}
}

func TestTodoDialog_MoveCardTargetStatus(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)

	status, ok := d.MoveCardTargetStatus(1)
	if !ok || status != session.TodoStatusInProgress {
		t.Errorf("expected InProgress moving right from Todo, got %s ok=%v", status, ok)
	}
	_, ok = d.MoveCardTargetStatus(-1)
	if ok {
		t.Error("expected no-op when moving left from Todo column (leftmost)")
	}
}

func TestTodoDialog_MoveCardTargetStatus_OrphanedGuard(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	// Board with a Done todo and an Orphaned todo → 5 columns (col 4 = orphaned)
	todos := []*session.Todo{
		makeTodo("a", session.TodoStatusDone),
		makeTodo("b", session.TodoStatusOrphaned),
	}
	d.Show("/proj", "", "", todos)
	// Position at Done column (col 3)
	d.selectedCol = 3

	// Moving right from Done would land on orphaned (col 4) → must be blocked
	_, ok := d.MoveCardTargetStatus(1)
	if ok {
		t.Error("expected move right from Done to be blocked (orphaned column)")
	}

	// Moving left from Done → InReview (col 2) → must be allowed
	status, ok := d.MoveCardTargetStatus(-1)
	if !ok || status != session.TodoStatusInReview {
		t.Errorf("expected InReview moving left from Done, got %s ok=%v", status, ok)
	}
}

func TestTodoDialog_HandleKey_ShiftRight_ReturnsMoveCardRight(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/proj", "", "", todos)
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
	action := d.HandleKey(tea.KeyMsg{Type: tea.KeyShiftLeft})
	if action != TodoActionNone {
		t.Errorf("expected TodoActionNone at left boundary, got %v", action)
	}
}

// keyMsg is a test helper that creates a tea.KeyMsg from a string.
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

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
	// Non-empty board with no orphaned todos — orphaned column should not appear
	todos := []*session.Todo{makeTodo("a", session.TodoStatusTodo)}
	d.Show("/myproject", "", "", todos)

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

func TestTodoDialog_ViewKanban_TruncatesLongTitle(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(80, 40) // narrow width to force truncation
	longTitle := strings.Repeat("a", 200)
	todos := []*session.Todo{makeTodo(longTitle, session.TodoStatusTodo)}
	d.Show("/myproject", "", "", todos)

	// Should not panic, and should not contain the full 200-char title
	view := d.View()
	if strings.Contains(view, longTitle) {
		t.Error("expected long title to be truncated in view")
	}
}

func TestTodoDialog_ViewKanban_TruncatesUnicodeTitle(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(80, 40)
	// Unicode title — must not panic on truncation
	unicodeTitle := strings.Repeat("日本語テスト", 30) // CJK chars, multibyte
	todos := []*session.Todo{makeTodo(unicodeTitle, session.TodoStatusTodo)}
	d.Show("/myproject", "", "", todos)

	// Must not panic — this is the primary assertion
	view := d.View()
	if strings.Contains(view, unicodeTitle) {
		t.Error("expected unicode title to be truncated in view")
	}
}

func TestRenderDetailPanel(t *testing.T) {
	t.Run("no selected todo returns empty string", func(t *testing.T) {
		d := NewTodoDialog()
		d.SetSize(80, 40)
		// No Show() called — SelectedTodo() returns nil
		got := d.renderDetailPanel(70)
		if got != "" {
			t.Errorf("expected empty string for nil selection, got %q", got)
		}
	})

	t.Run("shows description text when present", func(t *testing.T) {
		d := NewTodoDialog()
		d.SetSize(80, 40)
		todos := []*session.Todo{
			{ID: "1", Title: "My Task", Description: "This is the description", Status: session.TodoStatusTodo},
		}
		d.Show("/proj", "/proj", "proj", todos)
		got := d.renderDetailPanel(70)
		if !strings.Contains(got, "This is the description") {
			t.Errorf("expected description text in panel, got %q", got)
		}
	})

	t.Run("shows placeholder when description is empty", func(t *testing.T) {
		d := NewTodoDialog()
		d.SetSize(80, 40)
		todos := []*session.Todo{
			{ID: "1", Title: "No Desc", Description: "", Status: session.TodoStatusTodo},
		}
		d.Show("/proj", "/proj", "proj", todos)
		got := d.renderDetailPanel(70)
		if !strings.Contains(got, "no description") {
			t.Errorf("expected 'no description' placeholder in panel, got %q", got)
		}
	})
}

func TestTodoDialog_ViewKanban_ShowsDescriptionPanel(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	todos := []*session.Todo{
		{ID: "1", Title: "My Task", Description: "Important context here", Status: session.TodoStatusTodo},
	}
	d.Show("/proj", "/proj", "proj", todos)

	view := d.View()
	if !strings.Contains(view, "Important context here") {
		t.Errorf("expected description text in view:\n%s", view)
	}
	if !strings.Contains(view, "description") {
		t.Errorf("expected 'description' label in view:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_NoDescriptionShowsPlaceholder(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	todos := []*session.Todo{
		{ID: "1", Title: "No Desc Task", Description: "", Status: session.TodoStatusTodo},
	}
	d.Show("/proj", "/proj", "proj", todos)

	view := d.View()
	if !strings.Contains(view, "no description") {
		t.Errorf("expected 'no description' placeholder in view:\n%s", view)
	}
}

func TestTodoDialog_ViewKanban_NoDetailPanelOnEmptyColumn(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(160, 40)
	// Only a todo-column todo exists; in-progress (col 1) is empty
	todos := []*session.Todo{
		{ID: "1", Title: "A task", Description: "some desc", Status: session.TodoStatusTodo},
	}
	d.Show("/proj", "/proj", "proj", todos)
	// Move cursor to the empty in-progress column
	d.selectedCol = 1

	view := d.View()
	// Panel should be hidden — no placeholder, no description text
	if strings.Contains(view, "no description") {
		t.Errorf("expected no detail panel when cursor is on empty column, but got 'no description' in view")
	}
	if strings.Contains(view, "some desc") {
		t.Errorf("expected no description text when cursor is on empty column")
	}
}

func TestTodoDialog_PromptField_TabCycle(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	d.HandleKey(keyMsg("n")) // open new form

	// Initially focused on title (formFocus == 0)
	if d.formFocus != 0 {
		t.Fatalf("expected formFocus=0 after opening form, got %d", d.formFocus)
	}

	// Tab → desc
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	if d.formFocus != 1 {
		t.Errorf("expected formFocus=1 after first tab, got %d", d.formFocus)
	}

	// Tab → prompt
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	if d.formFocus != 2 {
		t.Errorf("expected formFocus=2 after second tab, got %d", d.formFocus)
	}

	// Tab → wraps back to title
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	if d.formFocus != 0 {
		t.Errorf("expected formFocus=0 after third tab (wrap), got %d", d.formFocus)
	}
}

func TestTodoDialog_PromptField_ShiftTabCycle(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	d.HandleKey(keyMsg("n"))

	// shift+tab from title wraps to prompt
	d.HandleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if d.formFocus != 2 {
		t.Errorf("expected formFocus=2 after shift+tab from title, got %d", d.formFocus)
	}
}

func TestTodoDialog_PromptField_GetFormValues(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)
	d.HandleKey(keyMsg("n"))

	// Type into title
	for _, ch := range "My Task" {
		d.HandleKey(keyMsg(string(ch)))
	}

	// Tab to prompt field
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})

	// Type into prompt
	for _, ch := range "implement hello world" {
		d.HandleKey(keyMsg(string(ch)))
	}

	title, _, prompt, _, _ := d.GetFormValues()
	if title != "My Task" {
		t.Errorf("expected title='My Task', got %q", title)
	}
	if prompt != "implement hello world" {
		t.Errorf("expected prompt='implement hello world', got %q", prompt)
	}
}

func TestTodoDialog_PromptField_OpenEditForm_Populates(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)

	todo := &session.Todo{
		ID:          "t1",
		Title:       "Existing Task",
		Description: "some desc",
		Prompt:      "do the thing",
		Status:      session.TodoStatusTodo,
	}

	d.openEditForm(todo)

	_, _, prompt, editingID, _ := d.GetFormValues()
	if prompt != "do the thing" {
		t.Errorf("expected prompt='do the thing', got %q", prompt)
	}
	if editingID != "t1" {
		t.Errorf("expected editingID='t1', got %q", editingID)
	}
}

func TestTodoDialog_PromptField_OpenNewForm_Clears(t *testing.T) {
	d := NewTodoDialog()
	d.SetSize(120, 40)
	d.Show("/proj", "", "", nil)

	// First open: type a prompt
	d.HandleKey(keyMsg("n"))
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	d.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	for _, ch := range "old prompt" {
		d.HandleKey(keyMsg(string(ch)))
	}

	// Escape back to kanban, open form again
	d.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	d.HandleKey(keyMsg("n"))

	_, _, prompt, _, _ := d.GetFormValues()
	if prompt != "" {
		t.Errorf("expected empty prompt after re-opening new form, got %q", prompt)
	}
}

func TestWordWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		maxLines int
		want     string
	}{
		{
			name:     "short text fits on one line",
			text:     "hello world",
			width:    20,
			maxLines: 3,
			want:     "hello world",
		},
		{
			name:     "wraps within maxLines",
			text:     "the quick brown",
			width:    10,
			maxLines: 3,
			want:     "the quick\nbrown",
		},
		{
			name:     "appends ellipsis when truncated at maxLines",
			text:     "ab cd ef gh ij kl",
			width:    6,
			maxLines: 2,
			want:     "ab cd\nef gh…",
		},
		{
			name:     "truncates last line chars when too long",
			text:     "abcdefgh ijklmnop qrst",
			width:    8,
			maxLines: 2,
			want:     "abcdefgh\nijklmno…",
		},
		{
			name:     "empty text",
			text:     "",
			width:    20,
			maxLines: 3,
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrapText(tt.text, tt.width, tt.maxLines)
			if got != tt.want {
				t.Errorf("wordWrapText(%q, %d, %d)\ngot:  %q\nwant: %q",
					tt.text, tt.width, tt.maxLines, got, tt.want)
			}
		})
	}
}
