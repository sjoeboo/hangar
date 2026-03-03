package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sjoeboo/hangar/internal/git"
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
	var sendErr error
	if req.Raw {
		sendErr = ts.SendKeys(req.Message)
	} else {
		sendErr = ts.SendKeysAndEnter(req.Message)
	}
	if sendErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("send failed: %v", sendErr))
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
	ts := inst.GetTmuxSession()
	if ts == nil {
		writeJSON(w, http.StatusOK, SessionOutputResponse{
			SessionID: inst.ID,
			Output:    "",
			Lines:     0,
		})
		return
	}
	// Support ?width= to capture at a specific terminal width (for web UI).
	var content string
	if widthStr := r.URL.Query().Get("width"); widthStr != "" {
		if w, err := strconv.Atoi(widthStr); err == nil && w > 0 && w <= 500 {
			content, _ = ts.CapturePaneWithWidth(w)
		}
	}
	if content == "" {
		var err error
		content, err = ts.CapturePane()
		if err != nil {
			// Fall back to latest prompt if capture fails
			writeJSON(w, http.StatusOK, SessionOutputResponse{
				SessionID: inst.ID,
				Output:    inst.LatestPrompt,
				Lines:     1,
			})
			return
		}
	}
	// Post-process: trim trailing spaces per line (removes tmux's pane-width
	// padding, e.g. 220-col padding on a wide server terminal), then join with
	// \r\n so xterm.js (convertEol:false) renders lines correctly.
	rawLines := strings.Split(content, "\n")
	for i, line := range rawLines {
		rawLines[i] = strings.TrimRight(line, " ")
	}
	content = strings.Join(rawLines, "\r\n")

	lines := strings.Count(content, "\r\n")
	writeJSON(w, http.StatusOK, SessionOutputResponse{
		SessionID: inst.ID,
		Output:    content,
		Lines:     lines,
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

	// Resolve the effective working directory before creating the instance.
	// For worktree sessions we must know the final path up front so the tmux
	// session is created with the correct WorkDir from the start.
	effectivePath := req.Path
	var worktreePath, worktreeBranch string
	if req.Worktree {
		worktreeBranch = req.Branch
		if worktreeBranch == "" {
			worktreeBranch = sanitizeBranchName(req.Title)
		}
		worktreePath = git.GenerateWorktreePath(req.Path, worktreeBranch, "subdirectory")

		// Pull latest base branch before branching (non-fatal on failure).
		defaultBranch, err := git.GetDefaultBranch(req.Path)
		if err != nil {
			slog.Warn("failed to get default branch", "err", err)
		} else {
			if err := git.UpdateBaseBranch(req.Path, defaultBranch); err != nil {
				slog.Warn("failed to update base branch", "err", err)
			}
		}

		if err := git.CreateWorktree(req.Path, worktreePath, worktreeBranch); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create worktree: %v", err))
			return
		}

		effectivePath = worktreePath
	}

	// Create the instance with the final working directory so the embedded
	// tmux session is initialised with the correct WorkDir from the start.
	inst := session.NewInstanceWithTool(req.Title, effectivePath, tool)
	// Command must be set so buildClaudeCommand (and siblings) know which
	// binary to run. For built-in tools the command name equals the tool name.
	inst.Command = tool
	if req.Group != "" {
		inst.GroupPath = req.Group
	}
	if req.ParentID != "" {
		inst.ParentSessionID = req.ParentID
	}
	if req.Worktree {
		inst.WorktreePath = worktreePath
		inst.WorktreeBranch = worktreeBranch
		inst.WorktreeRepoRoot = req.Path
	}

	// Handle skip-permissions
	if req.SkipPermissions {
		opts := inst.GetClaudeOptions()
		if opts == nil {
			opts = &session.ClaudeOptions{}
		}
		opts.SkipPermissions = true
		if err := inst.SetClaudeOptions(opts); err != nil {
			slog.Warn("failed to set skip_permissions", "err", err)
		}
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

	// Broadcast session_created event immediately so the frontend can show feedback.
	s.hub.broadcast <- WsMessage{Type: "session_created", Data: sessionToResponse(inst, s.getPRInfo)}

	// Trigger an immediate TUI reload so getInstances() returns the new session
	// without waiting for the 2-second StorageWatcher poll interval.
	if s.triggerReload != nil {
		s.triggerReload()
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.hub.broadcast <- WsMessage{Type: "sessions_changed"}
	}()

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
	// Kill tmux session and clean up worktree if applicable — mirrors the TUI's deleteSession.
	if inst := s.findInstance(id); inst != nil {
		if ts := inst.GetTmuxSession(); ts != nil {
			_ = ts.Kill()
		}
		if inst.IsWorktree() {
			_ = git.RemoveWorktree(inst.WorktreeRepoRoot, inst.WorktreePath, false)
			_ = git.PruneWorktrees(inst.WorktreeRepoRoot)
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

	// Trigger an immediate TUI reload so getInstances() reflects the deletion
	// without waiting for the 2-second StorageWatcher poll interval.
	if s.triggerReload != nil {
		s.triggerReload()
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.hub.broadcast <- WsMessage{Type: "sessions_changed"}
	}()
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

// sanitizeBranchName converts a session title into a valid git branch name.
func sanitizeBranchName(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return "session"
	}
	return string(result)
}
