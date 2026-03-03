package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sjoeboo/hangar/internal/session"
)

func projectToResponse(p *session.Project) ProjectResponse {
	return ProjectResponse{
		Name:       p.Name,
		BaseDir:    p.BaseDir,
		BaseBranch: p.BaseBranch,
		Order:      p.Order,
	}
}

// handleProjects routes GET /api/v1/projects and POST /api/v1/projects.
func (s *APIServer) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listProjects(w, r)
	case http.MethodPost:
		s.createProject(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProject routes GET/PATCH/DELETE /api/v1/projects/{id}.
// The {id} path parameter is the project Name (URL-encoded).
func (s *APIServer) handleProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		s.getProject(w, r, name)
	case http.MethodPatch:
		s.updateProject(w, r, name)
	case http.MethodDelete:
		s.deleteProject(w, r, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := session.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}
	resp := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, projectToResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *APIServer) getProject(w http.ResponseWriter, r *http.Request, name string) {
	projects, err := session.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}
	for _, p := range projects {
		if p.Name == name {
			writeJSON(w, http.StatusOK, projectToResponse(p))
			return
		}
	}
	writeError(w, http.StatusNotFound, "project not found")
}

func (s *APIServer) createProject(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req CreateProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.BaseDir == "" {
		writeError(w, http.StatusBadRequest, "name and base_dir are required")
		return
	}

	projects, err := session.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}
	// Duplicate check
	for _, p := range projects {
		if p.Name == req.Name {
			writeError(w, http.StatusConflict, "project already exists")
			return
		}
	}

	p := &session.Project{
		Name:       req.Name,
		BaseDir:    req.BaseDir,
		BaseBranch: req.BaseBranch,
		Order:      len(projects),
	}
	projects = append(projects, p)
	if err := session.SaveProjects(projects); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, projectToResponse(p))
}

func (s *APIServer) updateProject(w http.ResponseWriter, r *http.Request, name string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req UpdateProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	projects, err := session.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}

	var target *session.Project
	for _, p := range projects {
		if p.Name == name {
			target = p
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	if req.Name != nil {
		target.Name = strings.TrimSpace(*req.Name)
	}
	if req.BaseDir != nil {
		target.BaseDir = *req.BaseDir
	}
	if req.BaseBranch != nil {
		target.BaseBranch = *req.BaseBranch
	}

	if err := session.SaveProjects(projects); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, projectToResponse(target))
}

func (s *APIServer) deleteProject(w http.ResponseWriter, r *http.Request, name string) {
	projects, err := session.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("load error: %v", err))
		return
	}
	filtered := make([]*session.Project, 0, len(projects))
	found := false
	for _, p := range projects {
		if p.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}
	if !found {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := session.SaveProjects(filtered); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save error: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
