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
