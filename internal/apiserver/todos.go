package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sjoeboo/hangar/internal/session"
)

func todoToResponse(t *session.Todo) TodoResponse {
	return TodoResponse{
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

// handleTodos routes GET /api/v1/todos and POST /api/v1/todos.
// GET accepts ?project={path} query parameter to filter by project.
func (s *APIServer) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTodos(w, r)
	case http.MethodPost:
		s.createTodo(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTodo routes GET/PATCH/DELETE /api/v1/todos/{id}.
func (s *APIServer) handleTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		s.getTodo(w, r, id)
	case http.MethodPatch:
		s.updateTodo(w, r, id)
	case http.MethodDelete:
		s.deleteTodo(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) withTodoStorage(w http.ResponseWriter, fn func(*session.Storage)) {
	storage, err := session.NewStorageWithProfile(s.profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("storage error: %v", err))
		return
	}
	defer storage.Close()
	fn(storage)
}

func (s *APIServer) listTodos(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("project")
	if projectPath == "" {
		writeError(w, http.StatusBadRequest, "project query parameter required")
		return
	}
	s.withTodoStorage(w, func(storage *session.Storage) {
		todos, err := storage.LoadTodos(projectPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
			return
		}
		resp := make([]TodoResponse, 0, len(todos))
		for _, t := range todos {
			resp = append(resp, todoToResponse(t))
		}
		writeJSON(w, http.StatusOK, resp)
	})
}

func (s *APIServer) getTodo(w http.ResponseWriter, r *http.Request, id string) {
	s.withTodoStorage(w, func(storage *session.Storage) {
		t, err := storage.LoadTodoByID(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
			return
		}
		if t == nil {
			writeError(w, http.StatusNotFound, "todo not found")
			return
		}
		writeJSON(w, http.StatusOK, todoToResponse(t))
	})
}

func (s *APIServer) createTodo(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req CreateTodoRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" || req.ProjectPath == "" {
		writeError(w, http.StatusBadRequest, "title and project_path are required")
		return
	}
	todo := session.NewTodo(req.Title, req.Description, req.Prompt, req.ProjectPath)
	s.withTodoStorage(w, func(storage *session.Storage) {
		if err := storage.SaveTodo(todo); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
			return
		}
		resp := todoToResponse(todo)
		// Broadcast over WS
		s.hub.broadcast <- WsMessage{Type: "todo_updated", Data: resp}
		writeJSON(w, http.StatusCreated, resp)
	})
}

func (s *APIServer) updateTodo(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req UpdateTodoRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	s.withTodoStorage(w, func(storage *session.Storage) {
		t, err := storage.LoadTodoByID(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
			return
		}
		if t == nil {
			writeError(w, http.StatusNotFound, "todo not found")
			return
		}
		if req.Title != nil {
			t.Title = *req.Title
		}
		if req.Description != nil {
			t.Description = *req.Description
		}
		if req.Status != nil {
			t.Status = session.TodoStatus(*req.Status)
		}
		if req.SessionID != nil {
			t.SessionID = *req.SessionID
		}
		if err := storage.SaveTodo(t); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
			return
		}
		resp := todoToResponse(t)
		s.hub.broadcast <- WsMessage{Type: "todo_updated", Data: resp}
		writeJSON(w, http.StatusOK, resp)
	})
}

func (s *APIServer) deleteTodo(w http.ResponseWriter, r *http.Request, id string) {
	s.withTodoStorage(w, func(storage *session.Storage) {
		if err := storage.DeleteTodo(id); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("delete error: %v", err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
