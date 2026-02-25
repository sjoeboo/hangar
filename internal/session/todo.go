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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("storage database not initialized")
	}
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

// SaveTodo inserts or updates a todo. Updates todo.UpdatedAt to the current time before saving.
func (s *Storage) SaveTodo(todo *Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("storage database not initialized")
	}
	todo.UpdatedAt = time.Now()
	return s.db.SaveTodo(todoToRow(todo))
}

// DeleteTodo removes a todo by ID.
func (s *Storage) DeleteTodo(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("storage database not initialized")
	}
	return s.db.DeleteTodo(id)
}

// UpdateTodoStatus updates a todo's status and linked session ID.
func (s *Storage) UpdateTodoStatus(id string, status TodoStatus, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("storage database not initialized")
	}
	return s.db.UpdateTodoStatus(id, string(status), sessionID)
}

// OrphanTodosForSession sets status to "orphaned" for the todo linked to the given session.
// The todo-session relationship is 1:1 by design; at most one todo will be affected.
func (s *Storage) OrphanTodosForSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("storage database not initialized")
	}
	row, err := s.db.FindTodoBySessionID(sessionID)
	if err != nil || row == nil {
		return err
	}
	return s.db.UpdateTodoStatus(row.ID, "orphaned", sessionID)
}

// DeleteTodosForSession removes the todo linked to the given session.
// The todo-session relationship is 1:1 by design; at most one todo will be affected.
func (s *Storage) DeleteTodosForSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("storage database not initialized")
	}
	row, err := s.db.FindTodoBySessionID(sessionID)
	if err != nil || row == nil {
		return err
	}
	return s.db.DeleteTodo(row.ID)
}

// FindTodoBySessionID returns the todo linked to the given session, or nil.
func (s *Storage) FindTodoBySessionID(sessionID string) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("storage database not initialized")
	}
	row, err := s.db.FindTodoBySessionID(sessionID)
	if err != nil || row == nil {
		return nil, err
	}
	return todoFromRow(row), nil
}
