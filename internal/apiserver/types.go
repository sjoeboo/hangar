package apiserver

import "time"

// PRDashboardResponse is returned by GET /api/v1/prs.
type PRDashboardResponse struct {
	All             []*PRFullInfo          `json:"all"`
	Mine            []*PRFullInfo          `json:"mine"`
	ReviewRequested []*PRFullInfo          `json:"review_requested"`
	Sessions        map[string]*PRFullInfo `json:"sessions"` // keyed by sessionID
}

// PRFullInfo is the rich PR representation (superset of PRInfo).
type PRFullInfo struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	State          string    `json:"state"`
	IsDraft        bool      `json:"is_draft,omitempty"`
	URL            string    `json:"url"`
	Repo           string    `json:"repo,omitempty"`
	HeadBranch     string    `json:"head_branch,omitempty"`
	BaseBranch     string    `json:"base_branch,omitempty"`
	Author         string    `json:"author,omitempty"`
	ReviewDecision string    `json:"review_decision,omitempty"`
	CommentCount   int       `json:"comment_count,omitempty"`
	ChecksPassed   int       `json:"checks_passed,omitempty"`
	ChecksFailed   int       `json:"checks_failed,omitempty"`
	ChecksPending  int       `json:"checks_pending,omitempty"`
	HasChecks      bool      `json:"has_checks,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Source         string    `json:"source,omitempty"`     // "session", "mine", "review_requested"
	SessionID      string    `json:"session_id,omitempty"` // non-empty if linked to a session
}

// ReviewActionRequest is the JSON body for POST /api/v1/prs/review.
type ReviewActionRequest struct {
	Action string `json:"action"` // "approve", "request_changes", "comment"
	Body   string `json:"body,omitempty"`
}

// CommentRequest is the JSON body for POST /api/v1/prs/comment.
type CommentRequest struct {
	Body string `json:"body"`
	Path string `json:"path,omitempty"`
	Line int    `json:"line,omitempty"`
}

// StateActionRequest is the JSON body for POST /api/v1/prs/state.
type StateActionRequest struct {
	Action string `json:"action"` // "close", "reopen", "draft", "ready"
}

// SessionResponse is the JSON representation of a session returned by the API.
type SessionResponse struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	ProjectPath    string    `json:"project_path"`
	GroupPath      string    `json:"group_path"`
	SessionType    string    `json:"session_type,omitempty"`
	Tool           string    `json:"tool"`
	Status         string    `json:"status"`
	WorktreeBranch string    `json:"worktree_branch,omitempty"`
	LatestPrompt   string    `json:"latest_prompt,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at,omitempty"`
	ParentID       string    `json:"parent_id,omitempty"`
	PR             *PRInfo   `json:"pr,omitempty"` // nil if no PR or not a worktree session
}

// CreateSessionRequest is the JSON body for POST /api/v1/sessions.
type CreateSessionRequest struct {
	Title           string `json:"title"`
	Path            string `json:"path"`
	Tool            string `json:"tool,omitempty"`             // default: "claude"
	Group           string `json:"group,omitempty"`            // group path (e.g., "projects/myproject")
	ParentID        string `json:"parent_id,omitempty"`
	Message         string `json:"message,omitempty"`          // sent to session on start
	Worktree        bool   `json:"worktree,omitempty"`         // create git worktree for this session
	Branch          string `json:"branch,omitempty"`           // worktree branch name (auto-gen from title if empty)
	SkipPermissions bool   `json:"skip_permissions,omitempty"` // --dangerously-skip-permissions
}

// UpdateSessionRequest is the JSON body for PATCH /api/v1/sessions/{id}.
type UpdateSessionRequest struct {
	Title     *string `json:"title,omitempty"`
	GroupPath *string `json:"group_path,omitempty"`
	ParentID  *string `json:"parent_id,omitempty"`
}

// SendMessageRequest is the JSON body for POST /api/v1/sessions/{id}/send.
type SendMessageRequest struct {
	Message string `json:"message"`
	Raw     bool   `json:"raw,omitempty"` // if true, send literal keys without appending Enter
}

// StartSessionRequest is the optional JSON body for POST /api/v1/sessions/{id}/start.
type StartSessionRequest struct {
	Message string `json:"message,omitempty"` // optional initial message
}

// TodoResponse is the JSON representation of a todo returned by the API.
type TodoResponse struct {
	ID          string    `json:"id"`
	ProjectPath string    `json:"project_path"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	SessionID   string    `json:"session_id,omitempty"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTodoRequest is the JSON body for POST /api/v1/todos.
type CreateTodoRequest struct {
	ProjectPath string `json:"project_path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// UpdateTodoRequest is the JSON body for PATCH /api/v1/todos/{id}.
type UpdateTodoRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	SessionID   *string `json:"session_id,omitempty"`
}

// ProjectResponse is the JSON representation of a project returned by the API.
type ProjectResponse struct {
	Name       string `json:"name"`
	BaseDir    string `json:"base_dir"`
	BaseBranch string `json:"base_branch,omitempty"`
	Order      int    `json:"order,omitempty"`
}

// CreateProjectRequest is the JSON body for POST /api/v1/projects.
type CreateProjectRequest struct {
	Name       string `json:"name"`
	BaseDir    string `json:"base_dir"`
	BaseBranch string `json:"base_branch,omitempty"`
}

// UpdateProjectRequest is the JSON body for PATCH /api/v1/projects/{id}.
type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty"`
	BaseDir    *string `json:"base_dir,omitempty"`
	BaseBranch *string `json:"base_branch,omitempty"`
}

// StatusResponse is returned by GET /api/v1/status.
type StatusResponse struct {
	Version   string         `json:"version"`
	Uptime    string         `json:"uptime"`
	Sessions  int            `json:"sessions"`
	ByStatus  map[string]int `json:"by_status"`
}

// WsMessage is the envelope for all WebSocket messages (both directions).
type WsMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// WsSessionDeletedData is the data payload for session_deleted WS events.
type WsSessionDeletedData struct {
	ID string `json:"id"`
}

// WsHelloData is the data payload for the initial "hello" WS message.
type WsHelloData struct {
	Version  string `json:"version"`
	Sessions int    `json:"sessions"`
}

// SessionOutputResponse is returned by GET /api/v1/sessions/{id}/output.
type SessionOutputResponse struct {
	SessionID string `json:"session_id"`
	Output    string `json:"output"`
	Lines     int    `json:"lines"`
}

// SessionOutputData is the WS event payload for session_output events.
type SessionOutputData struct {
	SessionID string `json:"session_id"`
	Output    string `json:"output"`
}

// WsHookChangedData is the data payload for the hook_changed WS event.
type WsHookChangedData struct {
	InstanceID    string `json:"instance_id"`
	HookEventName string `json:"hook_event_name"`
	Status        string `json:"status"`
}

// PRInfo holds pull-request metadata for a session, sourced from the TUI's
// PR cache. All fields are omitempty so the object is absent when no PR exists.
type PRInfo struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	State         string `json:"state"` // OPEN, DRAFT, MERGED, CLOSED
	URL           string `json:"url"`
	ChecksPassed  int    `json:"checks_passed,omitempty"`
	ChecksFailed  int    `json:"checks_failed,omitempty"`
	ChecksPending int    `json:"checks_pending,omitempty"`
	HasChecks     bool   `json:"has_checks,omitempty"`
}
