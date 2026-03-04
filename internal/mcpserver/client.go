package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps the Hangar REST API.
type Client struct {
	base string
	http *http.Client
}

// NewClient creates a client pointing at the given base URL.
func NewClient(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// get performs a GET request and decodes the JSON response into v.
func (c *Client) get(path string, v any) error {
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, v)
}

// post performs a POST request with a JSON body and decodes the response into v (v may be nil).
func (c *Client) post(path string, body any, v any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.http.Post(c.base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("POST %s: HTTP %d: %s", path, resp.StatusCode, string(rb))
	}
	if v != nil {
		return json.Unmarshal(rb, v)
	}
	return nil
}

// patch performs a PATCH request.
func (c *Client) patch(path string, body any, v any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("PATCH %s: HTTP %d: %s", path, resp.StatusCode, string(rb))
	}
	if v != nil {
		return json.Unmarshal(rb, v)
	}
	return nil
}

// del performs a DELETE request.
func (c *Client) del(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE %s: HTTP %d: %s", path, resp.StatusCode, string(b))
	}
	return nil
}

// ListSessions returns all sessions.
func (c *Client) ListSessions() ([]map[string]any, error) {
	var result []map[string]any
	err := c.get("/api/v1/sessions", &result)
	return result, err
}

// GetSession returns a single session by ID.
func (c *Client) GetSession(id string) (map[string]any, error) {
	var result map[string]any
	err := c.get("/api/v1/sessions/"+id, &result)
	return result, err
}

// GetSessionOutput returns recent output lines for a session.
func (c *Client) GetSessionOutput(id string, lines int) (map[string]any, error) {
	var result map[string]any
	path := fmt.Sprintf("/api/v1/sessions/%s/output", id)
	if lines > 0 {
		path += fmt.Sprintf("?lines=%d", lines)
	}
	err := c.get(path, &result)
	return result, err
}

// SendMessage sends text to a running session.
func (c *Client) SendMessage(id, message string) error {
	body := map[string]string{"text": message}
	return c.post("/api/v1/sessions/"+id+"/send", body, nil)
}

// StartSession starts a session (with optional initial message).
func (c *Client) StartSession(id, message string) error {
	body := map[string]string{}
	if message != "" {
		body["message"] = message
	}
	return c.post("/api/v1/sessions/"+id+"/start", body, nil)
}

// StopSession stops a session.
func (c *Client) StopSession(id string) error {
	return c.post("/api/v1/sessions/"+id+"/stop", nil, nil)
}

// RestartSession restarts a session.
func (c *Client) RestartSession(id string) error {
	return c.post("/api/v1/sessions/"+id+"/restart", nil, nil)
}

// CreateSession creates a new session.
func (c *Client) CreateSession(title, path, tool string) (map[string]any, error) {
	body := map[string]string{"title": title, "project_path": path}
	if tool != "" {
		body["tool"] = tool
	}
	var result map[string]any
	err := c.post("/api/v1/sessions", body, &result)
	return result, err
}

// ListTodos returns todos, optionally filtered by project.
func (c *Client) ListTodos(project string) ([]map[string]any, error) {
	path := "/api/v1/todos"
	if project != "" {
		path += "?project=" + project
	}
	var result []map[string]any
	err := c.get(path, &result)
	return result, err
}

// CreateTodo creates a new todo.
func (c *Client) CreateTodo(project, title, description string) (map[string]any, error) {
	body := map[string]string{
		"project_path": project,
		"title":        title,
		"description":  description,
	}
	var result map[string]any
	err := c.post("/api/v1/todos", body, &result)
	return result, err
}

// UpdateTodo updates a todo.
func (c *Client) UpdateTodo(id string, fields map[string]any) error {
	return c.patch("/api/v1/todos/"+id, fields, nil)
}

// DeleteTodo deletes a todo.
func (c *Client) DeleteTodo(id string) error {
	return c.del("/api/v1/todos/" + id)
}
