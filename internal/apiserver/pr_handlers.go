package apiserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	internalprs "github.com/sjoeboo/hangar/internal/pr"
)

// prFullInfoFromPR converts a *pr.PR to a *PRFullInfo for the dashboard response.
func prFullInfoFromPR(p *internalprs.PR) *PRFullInfo {
	if p == nil {
		return nil
	}
	source := ""
	switch p.Source {
	case internalprs.SourceSession:
		source = "session"
	case internalprs.SourceMine:
		source = "mine"
	case internalprs.SourceReviewRequested:
		source = "review_requested"
	}
	return &PRFullInfo{
		Number:         p.Number,
		Title:          p.Title,
		State:          p.State,
		IsDraft:        p.IsDraft,
		URL:            p.URL,
		Repo:           p.Repo,
		HeadBranch:     p.HeadBranch,
		BaseBranch:     p.BaseBranch,
		Author:         p.Author,
		ReviewDecision: p.ReviewDecision,
		CommentCount:   p.CommentCount,
		ChecksPassed:   p.ChecksPassed,
		ChecksFailed:   p.ChecksFailed,
		ChecksPending:  p.ChecksPending,
		HasChecks:      p.HasChecks,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		Source:         source,
		SessionID:      p.SessionID,
	}
}

// handlePRDashboard serves GET /api/v1/prs.
// Returns 503 if prManager is not configured.
func (s *APIServer) handlePRDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.prManager == nil {
		writeError(w, http.StatusServiceUnavailable, "PR manager not available")
		return
	}

	allPRs := s.prManager.GetAll()
	mine := s.prManager.GetMine()
	reviewRequested := s.prManager.GetReviewRequested()
	sessionPRs := s.prManager.GetSessionPRs()

	resp := PRDashboardResponse{
		All:             make([]*PRFullInfo, 0, len(allPRs)),
		Mine:            make([]*PRFullInfo, 0, len(mine)),
		ReviewRequested: make([]*PRFullInfo, 0, len(reviewRequested)),
		Sessions:        make(map[string]*PRFullInfo, len(sessionPRs)),
	}

	for _, p := range allPRs {
		resp.All = append(resp.All, prFullInfoFromPR(p))
	}
	for _, p := range mine {
		resp.Mine = append(resp.Mine, prFullInfoFromPR(p))
	}
	for _, p := range reviewRequested {
		resp.ReviewRequested = append(resp.ReviewRequested, prFullInfoFromPR(p))
	}
	for sessionID, p := range sessionPRs {
		if p != nil {
			resp.Sessions[sessionID] = prFullInfoFromPR(p)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// parsePRQueryParams extracts repo and number from query parameters.
// Returns ("", 0, false) and writes an error if required params are missing/invalid.
func parsePRQueryParams(w http.ResponseWriter, r *http.Request) (repo string, number int, ok bool) {
	repo = r.URL.Query().Get("repo")
	if repo == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: repo")
		return "", 0, false
	}
	numStr := r.URL.Query().Get("number")
	if numStr == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: number")
		return "", 0, false
	}
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		writeError(w, http.StatusBadRequest, "invalid query parameter: number")
		return "", 0, false
	}
	return repo, n, true
}

// handlePRDetail serves GET /api/v1/prs/detail?repo=owner%2Frepo&number=123.
// Lazily fetches full PR detail (comments, reviews, files, diff).
// Returns 503 if prManager is not configured.
func (s *APIServer) handlePRDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.prManager == nil {
		writeError(w, http.StatusServiceUnavailable, "PR manager not available")
		return
	}

	repo, number, ok := parsePRQueryParams(w, r)
	if !ok {
		return
	}

	detail, err := s.prManager.FetchDetail(repo, number)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		writeError(w, http.StatusNotFound, "PR not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// handlePRReview serves POST /api/v1/prs/review?repo=owner%2Frepo&number=123.
// Body: {"action": "approve"|"request_changes"|"comment", "body": "optional"}.
// Returns 503 if prManager is not configured.
func (s *APIServer) handlePRReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.prManager == nil {
		writeError(w, http.StatusServiceUnavailable, "PR manager not available")
		return
	}

	repo, number, ok := parsePRQueryParams(w, r)
	if !ok {
		return
	}

	var req ReviewActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	ghPath := s.prManager.GHPath()
	if ghPath == "" {
		writeError(w, http.StatusServiceUnavailable, "gh CLI not available")
		return
	}

	var err error
	switch req.Action {
	case "approve":
		err = internalprs.Approve(ghPath, repo, number, req.Body)
	case "request_changes":
		err = internalprs.RequestChanges(ghPath, repo, number, req.Body)
	case "comment":
		err = internalprs.AddComment(ghPath, repo, number, req.Body, "", 0)
	default:
		writeError(w, http.StatusBadRequest, "invalid action: must be approve, request_changes, or comment")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Invalidate cached detail so the next fetch is fresh.
	s.prManager.InvalidateDetail(repo, number)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handlePRComment serves POST /api/v1/prs/comment?repo=owner%2Frepo&number=123.
// Body: {"body": "...", "path": "optional/file.go", "line": 0}.
// Returns 503 if prManager is not configured.
func (s *APIServer) handlePRComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.prManager == nil {
		writeError(w, http.StatusServiceUnavailable, "PR manager not available")
		return
	}

	repo, number, ok := parsePRQueryParams(w, r)
	if !ok {
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	ghPath := s.prManager.GHPath()
	if ghPath == "" {
		writeError(w, http.StatusServiceUnavailable, "gh CLI not available")
		return
	}

	if err := internalprs.AddComment(ghPath, repo, number, req.Body, req.Path, req.Line); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Invalidate cached detail so the next fetch is fresh.
	s.prManager.InvalidateDetail(repo, number)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handlePRState serves POST /api/v1/prs/state?repo=owner%2Frepo&number=123.
// Body: {"action": "close"|"reopen"|"draft"|"ready"}.
// Returns 503 if prManager is not configured.
func (s *APIServer) handlePRState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.prManager == nil {
		writeError(w, http.StatusServiceUnavailable, "PR manager not available")
		return
	}

	repo, number, ok := parsePRQueryParams(w, r)
	if !ok {
		return
	}

	var req StateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	ghPath := s.prManager.GHPath()
	if ghPath == "" {
		writeError(w, http.StatusServiceUnavailable, "gh CLI not available")
		return
	}

	var err error
	switch req.Action {
	case "close":
		err = internalprs.Close(ghPath, repo, number)
	case "reopen":
		err = internalprs.Reopen(ghPath, repo, number)
	case "draft":
		err = internalprs.ConvertToDraft(ghPath, repo, number)
	case "ready":
		err = internalprs.ConvertToReady(ghPath, repo, number)
	default:
		writeError(w, http.StatusBadRequest, "invalid action: must be close, reopen, draft, or ready")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Invalidate cached detail so the next fetch is fresh.
	s.prManager.InvalidateDetail(repo, number)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
