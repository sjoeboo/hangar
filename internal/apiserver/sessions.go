package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sjoeboo/hangar/internal/session"
)

// sessionToResponse converts an Instance to a SessionResponse DTO.
// getPRInfo is called when non-nil to populate PR data from the TUI cache.
func sessionToResponse(inst *session.Instance, getPRInfo func(string) *PRInfo) SessionResponse {
	resp := SessionResponse{
		ID:             inst.ID,
		Title:          inst.Title,
		ProjectPath:    inst.ProjectPath,
		GroupPath:      inst.GroupPath,
		Tool:           inst.Tool,
		Status:         string(inst.Status),
		WorktreeBranch: inst.WorktreeBranch,
		LatestPrompt:   inst.LatestPrompt,
		CreatedAt:      inst.CreatedAt,
		LastAccessedAt: inst.LastAccessedAt,
		ParentID:       inst.ParentSessionID,
	}
	if getPRInfo != nil {
		resp.PR = getPRInfo(inst.ID)
	}
	return resp
}

// findInstance returns the live instance with the given ID, or nil.
func (s *APIServer) findInstance(id string) *session.Instance {
	for _, inst := range s.instances() {
		if inst.ID == id {
			return inst
		}
	}
	return nil
}

// handleSessions routes GET /api/v1/sessions and POST /api/v1/sessions.
func (s *APIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSessions(w, r)
	case http.MethodPost:
		s.createSession(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSession routes GET/PATCH/DELETE /api/v1/sessions/{id}.
func (s *APIServer) handleSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		s.getSession(w, r, id)
	case http.MethodPatch:
		s.updateSession(w, r, id)
	case http.MethodDelete:
		s.deleteSession(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleSessionStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	var req StartSessionRequest
	if body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)); err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	if err := inst.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("start failed: %v", err))
		return
	}
	if req.Message != "" {
		ts := inst.GetTmuxSession()
		if ts != nil {
			_ = ts.SendKeysAndEnter(req.Message)
		}
	}
	writeJSON(w, http.StatusOK, sessionToResponse(inst, s.getPRInfo))
}

func (s *APIServer) handleSessionStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	ts := inst.GetTmuxSession()
	if ts == nil {
		writeError(w, http.StatusConflict, "session has no tmux session")
		return
	}
	if err := ts.Kill(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("stop failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *APIServer) handleSessionRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if err := inst.Restart(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("restart failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, sessionToResponse(inst, s.getPRInfo))
}

func (s *APIServer) handleSessionSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req SendMessageRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Message == "" {
		writeError(w, http.StatusBadRequest, "message field required")
		return
	}
	ts := inst.GetTmuxSession()
	if ts == nil {
		writeError(w, http.StatusConflict, "session has no tmux session")
		return
	}
	if err := ts.SendKeysAndEnter(req.Message); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("send failed: %v", err))
		return
	}
	// Broadcast update over WS
	s.hub.broadcast <- WsMessage{Type: "session_updated", Data: sessionToResponse(inst, s.getPRInfo)}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *APIServer) handleSessionOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"session_id":    inst.ID,
		"latest_prompt": inst.LatestPrompt,
	})
}

// listSessions handles GET /api/v1/sessions.
func (s *APIServer) listSessions(w http.ResponseWriter, r *http.Request) {
	instances := s.instances()
	resp := make([]SessionResponse, 0, len(instances))
	for _, inst := range instances {
		resp = append(resp, sessionToResponse(inst, s.getPRInfo))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getSession handles GET /api/v1/sessions/{id}.
func (s *APIServer) getSession(w http.ResponseWriter, r *http.Request, id string) {
	inst := s.findInstance(id)
	if inst == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sessionToResponse(inst, s.getPRInfo))
}

// createSession handles POST /api/v1/sessions.
// Creates and immediately starts a new session.
func (s *APIServer) createSession(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req CreateSessionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, "title and path are required")
		return
	}
	tool := req.Tool
	if tool == "" {
		tool = "claude"
	}

	inst := session.NewInstanceWithTool(req.Title, req.Path, tool)
	if req.Group != "" {
		inst.GroupPath = req.Group
	}
	if req.ParentID != "" {
		inst.ParentSessionID = req.ParentID
	}

	// Persist to storage before starting so the TUI picks it up
	storage, err := session.NewStorageWithProfile(s.profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("storage error: %v", err))
		return
	}
	defer storage.Close()

	existing, err := storage.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}
	all := append(existing, inst)
	if err := storage.Save(all); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
		return
	}

	// Start the tmux session
	if err := inst.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("start error: %v", err))
		return
	}

	// Send initial message if provided
	if req.Message != "" {
		if ts := inst.GetTmuxSession(); ts != nil {
			_ = ts.SendKeysAndEnter(req.Message)
		}
	}

	// Broadcast session_created event
	s.hub.broadcast <- WsMessage{Type: "session_created", Data: sessionToResponse(inst, s.getPRInfo)}

	writeJSON(w, http.StatusCreated, sessionToResponse(inst, s.getPRInfo))
}

// updateSession handles PATCH /api/v1/sessions/{id}.
func (s *APIServer) updateSession(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req UpdateSessionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	storage, err := session.NewStorageWithProfile(s.profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("storage error: %v", err))
		return
	}
	defer storage.Close()

	instances, err := storage.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}

	var target *session.Instance
	for _, inst := range instances {
		if inst.ID == id {
			target = inst
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if req.Title != nil {
		target.Title = strings.TrimSpace(*req.Title)
	}
	if req.GroupPath != nil {
		target.GroupPath = *req.GroupPath
	}
	if req.ParentID != nil {
		target.ParentSessionID = *req.ParentID
	}

	if err := storage.Save(instances); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, sessionToResponse(target, s.getPRInfo))
}

// deleteSession handles DELETE /api/v1/sessions/{id}.
func (s *APIServer) deleteSession(w http.ResponseWriter, r *http.Request, id string) {
	// Kill tmux session if it's live
	if inst := s.findInstance(id); inst != nil {
		if ts := inst.GetTmuxSession(); ts != nil {
			_ = ts.Kill()
		}
	}

	storage, err := session.NewStorageWithProfile(s.profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("storage error: %v", err))
		return
	}
	defer storage.Close()

	if err := storage.DeleteInstance(id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("delete error: %v", err))
		return
	}

	s.hub.broadcast <- WsMessage{Type: "session_deleted", Data: WsSessionDeletedData{ID: id}}
	w.WriteHeader(http.StatusNoContent)
}

// wsHandleSendMessage handles send_message commands from WS clients.
func (s *APIServer) wsHandleSendMessage(c *Client, msg WsMessage) {
	raw, err := json.Marshal(msg.Data)
	if err != nil {
		return
	}
	var data struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || data.SessionID == "" || data.Message == "" {
		return
	}
	inst := s.findInstance(data.SessionID)
	if inst == nil {
		return
	}
	if ts := inst.GetTmuxSession(); ts != nil {
		_ = ts.SendKeysAndEnter(data.Message)
		s.hub.broadcast <- WsMessage{Type: "session_updated", Data: sessionToResponse(inst, s.getPRInfo)}
	}
}

// wsHandleStopSession handles stop_session commands from WS clients.
func (s *APIServer) wsHandleStopSession(c *Client, msg WsMessage) {
	raw, err := json.Marshal(msg.Data)
	if err != nil {
		return
	}
	var data struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || data.SessionID == "" {
		return
	}
	inst := s.findInstance(data.SessionID)
	if inst == nil {
		return
	}
	if ts := inst.GetTmuxSession(); ts != nil {
		_ = ts.Kill()
		s.hub.broadcast <- WsMessage{Type: "session_updated", Data: sessionToResponse(inst, s.getPRInfo)}
	}
}
