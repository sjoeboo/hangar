package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/tmux"
)

// Status represents the current state of a session
type Status string

const (
	StatusRunning Status = "running"
	StatusWaiting Status = "waiting"
	StatusIdle    Status = "idle"
	StatusError   Status = "error"
)

// Instance represents a single agent/shell session
type Instance struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	ProjectPath    string    `json:"project_path"`
	GroupPath      string    `json:"group_path"` // e.g., "projects/devops"
	Command        string    `json:"command"`
	Tool           string    `json:"tool"`
	Status         Status    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at,omitempty"` // When user last attached

	// Claude Code integration
	ClaudeSessionID  string    `json:"claude_session_id,omitempty"`
	ClaudeDetectedAt time.Time `json:"claude_detected_at,omitempty"`

	// MCP tracking - which MCPs were loaded when session started/restarted
	// Used to detect pending MCPs (added after session start) and stale MCPs (removed but still running)
	LoadedMCPNames []string `json:"loaded_mcp_names,omitempty"`

	tmuxSession *tmux.Session // Internal tmux session

	// lastErrorCheck tracks when we last confirmed the session doesn't exist
	// Used to skip expensive Exists() checks for ghost sessions (sessions in JSON but not in tmux)
	// Not serialized - resets on load, but that's fine since we'll recheck on first poll
	lastErrorCheck time.Time
}

// MarkAccessed updates the LastAccessedAt timestamp to now
func (inst *Instance) MarkAccessed() {
	inst.LastAccessedAt = time.Now()
}

// GetLastActivityTime returns when the session was last active (content changed)
// Returns CreatedAt if no activity has been tracked yet
func (inst *Instance) GetLastActivityTime() time.Time {
	if inst.tmuxSession != nil {
		activityTime := inst.tmuxSession.GetLastActivityTime()
		if !activityTime.IsZero() {
			return activityTime
		}
	}
	// Fallback to CreatedAt
	return inst.CreatedAt
}

// NewInstance creates a new session instance
func NewInstance(title, projectPath string) *Instance {
	return &Instance{
		ID:          generateID(),
		Title:       title,
		ProjectPath: projectPath,
		GroupPath:   extractGroupPath(projectPath), // Auto-assign group from path
		Tool:        "shell",
		Status:      StatusIdle,
		CreatedAt:   time.Now(),
		tmuxSession: tmux.NewSession(title, projectPath),
	}
}

// NewInstanceWithGroup creates a new session instance with explicit group
func NewInstanceWithGroup(title, projectPath, groupPath string) *Instance {
	inst := NewInstance(title, projectPath)
	inst.GroupPath = groupPath
	return inst
}

// NewInstanceWithTool creates a new session with tool-specific initialization
func NewInstanceWithTool(title, projectPath, tool string) *Instance {
	inst := &Instance{
		ID:          generateID(),
		Title:       title,
		ProjectPath: projectPath,
		GroupPath:   extractGroupPath(projectPath),
		Tool:        tool,
		Status:      StatusIdle,
		CreatedAt:   time.Now(),
		tmuxSession: tmux.NewSession(title, projectPath),
	}

	// Claude session ID will be detected from files Claude creates
	// No pre-assignment needed

	return inst
}

// NewInstanceWithGroupAndTool creates a new session with explicit group and tool
func NewInstanceWithGroupAndTool(title, projectPath, groupPath, tool string) *Instance {
	inst := NewInstanceWithTool(title, projectPath, tool)
	inst.GroupPath = groupPath
	return inst
}

// extractGroupPath extracts a group path from project path
// e.g., "/home/user/projects/devops" -> "projects"
func extractGroupPath(projectPath string) string {
	parts := strings.Split(projectPath, "/")
	// Find meaningful directory (skip Users, home, etc.)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part != "" && part != "Users" && part != "home" && !strings.HasPrefix(part, ".") {
			// Return parent directory as group if we're at project level
			if i > 0 && i == len(parts)-1 {
				parent := parts[i-1]
				if parent != "" && parent != "Users" && parent != "home" && !strings.HasPrefix(parent, ".") {
					return parent
				}
			}
			return part
		}
	}
	return DefaultGroupName
}

// buildClaudeCommand builds the claude command with session capture
// For new sessions: captures session ID via print mode, stores in tmux env, then resumes
// This ensures we always know the session ID for fork/restart features
// Respects: CLAUDE_CONFIG_DIR, dangerous_mode from user config
func (i *Instance) buildClaudeCommand(baseCommand string) string {
	if i.Tool != "claude" {
		return baseCommand
	}

	configDir := GetClaudeConfigDir()

	// Check if dangerous mode is enabled in user config
	dangerousMode := false
	if userConfig, err := LoadUserConfig(); err == nil && userConfig != nil {
		dangerousMode = userConfig.Claude.DangerousMode
	}

	// If baseCommand is just "claude", build the capture-resume command
	// This command:
	// 1. Starts Claude in print mode to get session ID
	// 2. Stores session ID in tmux environment (for retrieval by agent-deck)
	// 3. Resumes that session interactively (with dangerous mode if enabled)
	if baseCommand == "claude" {
		if dangerousMode {
			return fmt.Sprintf(
				`session_id=$(CLAUDE_CONFIG_DIR=%s claude -p "." --output-format json 2>/dev/null | jq -r '.session_id') && `+
					`tmux set-environment CLAUDE_SESSION_ID "$session_id" && `+
					`CLAUDE_CONFIG_DIR=%s claude --resume "$session_id" --dangerously-skip-permissions`,
				configDir, configDir)
		}
		return fmt.Sprintf(
			`session_id=$(CLAUDE_CONFIG_DIR=%s claude -p "." --output-format json 2>/dev/null | jq -r '.session_id') && `+
				`tmux set-environment CLAUDE_SESSION_ID "$session_id" && `+
				`CLAUDE_CONFIG_DIR=%s claude --resume "$session_id"`,
			configDir, configDir)
	}

	// For custom commands (e.g., fork commands), return as-is
	return baseCommand
}

// Start starts the session in tmux
func (i *Instance) Start() error {
	if i.tmuxSession == nil {
		return fmt.Errorf("tmux session not initialized")
	}

	// Build command (adds config dir for claude)
	command := i.buildClaudeCommand(i.Command)

	// Start the tmux session
	if err := i.tmuxSession.Start(command); err != nil {
		return fmt.Errorf("failed to start tmux session: %w", err)
	}

	// Capture MCPs that are now loaded (for sync tracking)
	i.CaptureLoadedMCPs()

	if command != "" {
		i.Status = StatusRunning
	}

	return nil
}

// errorRecheckInterval - how often to recheck sessions that don't exist
// Ghost sessions (in JSON but not in tmux) are rechecked at this interval
// instead of every 500ms tick, dramatically reducing subprocess spawns
const errorRecheckInterval = 30 * time.Second

// UpdateStatus updates the session status by checking tmux
func (i *Instance) UpdateStatus() error {
	if i.tmuxSession == nil {
		i.Status = StatusError
		return nil
	}

	// Optimization: Skip expensive Exists() check for sessions already in error status
	// Ghost sessions (in JSON but not in tmux) only get rechecked every 30 seconds
	// This reduces subprocess spawns from 74/sec to ~5/sec for 28 ghost sessions
	if i.Status == StatusError && !i.lastErrorCheck.IsZero() &&
		time.Since(i.lastErrorCheck) < errorRecheckInterval {
		return nil // Skip - still in error, checked recently
	}

	// Check if tmux session exists
	if !i.tmuxSession.Exists() {
		i.Status = StatusError
		i.lastErrorCheck = time.Now() // Record when we confirmed error
		return nil
	}

	// Session exists - clear error check timestamp
	i.lastErrorCheck = time.Time{}

	// Get status from tmux session
	status, err := i.tmuxSession.GetStatus()
	if err != nil {
		i.Status = StatusError
		return err
	}

	// Map tmux status to instance status
	switch status {
	case "active":
		i.Status = StatusRunning
	case "waiting":
		i.Status = StatusWaiting
	case "idle":
		i.Status = StatusIdle
	default:
		i.Status = StatusError
	}

	// Update tool detection dynamically (enables fork when Claude starts)
	if detectedTool := i.tmuxSession.DetectTool(); detectedTool != "" {
		i.Tool = detectedTool
	}

	// Update Claude session tracking (non-blocking, best-effort)
	// Pass nil for excludeIDs - deduplication happens at manager level
	i.UpdateClaudeSession(nil)

	return nil
}

// UpdateClaudeSession updates the Claude session ID using detection
// Priority: 1) tmux environment (for sessions we started), 2) file scanning (legacy/imported)
// excludeIDs contains session IDs already claimed by other instances
// Pass nil to skip deduplication (when called from UpdateStatus)
func (i *Instance) UpdateClaudeSession(excludeIDs map[string]bool) {
	if i.Tool != "claude" {
		return
	}

	// If we already have a session ID and it's recent, just refresh timestamp
	if i.ClaudeSessionID != "" && time.Since(i.ClaudeDetectedAt) < 5*time.Minute {
		return
	}

	// PRIMARY: Try tmux environment first (most reliable for sessions we started)
	if sessionID := i.GetSessionIDFromTmux(); sessionID != "" {
		i.ClaudeSessionID = sessionID
		i.ClaudeDetectedAt = time.Now()
		return
	}

	// FALLBACK: File scanning (for imported/legacy sessions)
	workDir := i.ProjectPath
	if i.tmuxSession != nil {
		if wd := i.tmuxSession.GetWorkDir(); wd != "" {
			workDir = wd
		}
	}

	// Use the new FindSessionForInstance with timestamp filtering and deduplication
	sessionID := FindSessionForInstance(workDir, i.CreatedAt, excludeIDs)
	if sessionID != "" {
		i.ClaudeSessionID = sessionID
		i.ClaudeDetectedAt = time.Now()
	}
}

// WaitForClaudeSession waits for Claude to create a session file (for forked sessions)
// Returns the detected session ID or empty string after timeout
// Uses FindSessionForInstance with timestamp filtering to ensure we only detect
// session files created AFTER this instance started (not parent's pre-existing file)
func (i *Instance) WaitForClaudeSession(maxWait time.Duration) string {
	if i.Tool != "claude" {
		return ""
	}

	workDir := i.ProjectPath
	if i.tmuxSession != nil {
		if wd := i.tmuxSession.GetWorkDir(); wd != "" {
			workDir = wd
		}
	}

	// Poll every 200ms for up to maxWait
	interval := 200 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		// Use FindSessionForInstance with timestamp filtering
		// This ensures we only match files created AFTER this instance started
		// Critical for forks: prevents detecting parent's file instead of new fork file
		sessionID := FindSessionForInstance(workDir, i.CreatedAt, nil)
		if sessionID != "" {
			i.ClaudeSessionID = sessionID
			i.ClaudeDetectedAt = time.Now()
			return sessionID
		}
		time.Sleep(interval)
	}

	return ""
}

// WaitForClaudeSessionWithExclude waits for Claude to create a session file with exclusion list
// This is more robust than WaitForClaudeSession as it explicitly excludes known session IDs
// Use this when forking to ensure the fork's new session is detected, not an existing one
func (i *Instance) WaitForClaudeSessionWithExclude(maxWait time.Duration, excludeIDs map[string]bool) string {
	if i.Tool != "claude" {
		return ""
	}

	workDir := i.ProjectPath
	if i.tmuxSession != nil {
		if wd := i.tmuxSession.GetWorkDir(); wd != "" {
			workDir = wd
		}
	}

	// Poll every 200ms for up to maxWait
	interval := 200 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		// Use FindSessionForInstance with timestamp filtering AND exclusion list
		// This ensures we only match files:
		// 1. Created AFTER this instance started (timestamp filter)
		// 2. Not already claimed by another session (excludeIDs)
		sessionID := FindSessionForInstance(workDir, i.CreatedAt, excludeIDs)
		if sessionID != "" {
			i.ClaudeSessionID = sessionID
			i.ClaudeDetectedAt = time.Now()
			return sessionID
		}
		time.Sleep(interval)
	}

	return ""
}

// Preview returns the last 3 lines of terminal output
func (i *Instance) Preview() (string, error) {
	if i.tmuxSession == nil {
		return "", fmt.Errorf("tmux session not initialized")
	}

	content, err := i.tmuxSession.CapturePane()
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}

	return strings.Join(lines, "\n"), nil
}

// PreviewFull returns all terminal output
func (i *Instance) PreviewFull() (string, error) {
	if i.tmuxSession == nil {
		return "", fmt.Errorf("tmux session not initialized")
	}

	return i.tmuxSession.CaptureFullHistory()
}

// HasUpdated checks if there's new output since last check
func (i *Instance) HasUpdated() bool {
	if i.tmuxSession == nil {
		return false
	}

	updated, err := i.tmuxSession.HasUpdated()
	if err != nil {
		return false
	}

	return updated
}

// Kill terminates the tmux session
func (i *Instance) Kill() error {
	if i.tmuxSession == nil {
		return fmt.Errorf("tmux session not initialized")
	}

	if err := i.tmuxSession.Kill(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}
	i.Status = StatusError
	return nil
}

// Restart restarts the Claude session
// For Claude sessions with known ID: sends Ctrl+C twice and resume command to existing session
// For dead sessions or unknown ID: recreates the tmux session
func (i *Instance) Restart() error {
	log.Printf("[MCP-DEBUG] Instance.Restart() called - Tool=%s, ClaudeSessionID=%q, tmuxSession=%v, tmuxExists=%v",
		i.Tool, i.ClaudeSessionID, i.tmuxSession != nil, i.tmuxSession != nil && i.tmuxSession.Exists())

	// If Claude session with known ID AND tmux session exists, use respawn-pane
	if i.Tool == "claude" && i.ClaudeSessionID != "" && i.tmuxSession != nil && i.tmuxSession.Exists() {
		// Build the resume command with proper config
		resumeCmd := i.buildClaudeResumeCommand()
		log.Printf("[MCP-DEBUG] Using respawn-pane with command: %s", resumeCmd)

		// Use respawn-pane for atomic restart
		// This is more reliable than Ctrl+C + wait for shell + send command
		// respawn-pane -k kills the current process and starts the new command atomically
		if err := i.tmuxSession.RespawnPane(resumeCmd); err != nil {
			log.Printf("[MCP-DEBUG] RespawnPane failed: %v", err)
			return fmt.Errorf("failed to restart Claude session: %w", err)
		}

		log.Printf("[MCP-DEBUG] RespawnPane succeeded")

		// Re-capture MCPs after restart (they may have changed since session started)
		i.CaptureLoadedMCPs()

		i.Status = StatusRunning
		return nil
	}

	log.Printf("[MCP-DEBUG] Using fallback: recreate tmux session")

	// Fallback: recreate tmux session (for dead sessions or unknown ID)
	i.tmuxSession = tmux.NewSession(i.Title, i.ProjectPath)

	var command string
	if i.Tool == "claude" && i.ClaudeSessionID != "" {
		command = i.buildClaudeResumeCommand()
	} else {
		command = i.buildClaudeCommand(i.Command)
	}
	log.Printf("[MCP-DEBUG] Starting new tmux session with command: %s", command)

	if err := i.tmuxSession.Start(command); err != nil {
		log.Printf("[MCP-DEBUG] tmuxSession.Start() failed: %v", err)
		i.Status = StatusError
		return fmt.Errorf("failed to restart tmux session: %w", err)
	}

	log.Printf("[MCP-DEBUG] tmuxSession.Start() succeeded")

	// Re-capture MCPs after restart
	i.CaptureLoadedMCPs()

	if command != "" {
		i.Status = StatusRunning
	} else {
		i.Status = StatusIdle
	}

	return nil
}

// buildClaudeResumeCommand builds the claude resume command with proper config options
// Respects: CLAUDE_CONFIG_DIR, dangerous_mode from user config
func (i *Instance) buildClaudeResumeCommand() string {
	configDir := GetClaudeConfigDir()

	// Check if dangerous mode is enabled in user config
	dangerousMode := false
	if userConfig, err := LoadUserConfig(); err == nil && userConfig != nil {
		dangerousMode = userConfig.Claude.DangerousMode
	}

	// Build the command
	if dangerousMode {
		return fmt.Sprintf("CLAUDE_CONFIG_DIR=%s claude --resume %s --dangerously-skip-permissions",
			configDir, i.ClaudeSessionID)
	}
	return fmt.Sprintf("CLAUDE_CONFIG_DIR=%s claude --resume %s",
		configDir, i.ClaudeSessionID)
}

// CanRestart returns true if the session can be restarted
// For Claude sessions with known ID: can always restart (interrupt and resume)
// For other sessions: only if dead/error state
func (i *Instance) CanRestart() bool {
	// Claude sessions with known session ID can always be restarted
	if i.Tool == "claude" && i.ClaudeSessionID != "" {
		return true
	}
	// Other sessions: only if dead or error
	return i.Status == StatusError || i.tmuxSession == nil || !i.tmuxSession.Exists()
}

// CanFork returns true if this session can be forked (has recent Claude session)
func (i *Instance) CanFork() bool {
	if i.ClaudeSessionID == "" {
		return false
	}
	// Session ID must be detected within last 5 minutes
	return time.Since(i.ClaudeDetectedAt) < 5*time.Minute
}

// Fork returns the command to create a forked Claude session
// Uses capture-resume pattern: starts fork in print mode to get new session ID,
// stores in tmux environment, then resumes interactively
func (i *Instance) Fork(newTitle, newGroupPath string) (string, error) {
	if !i.CanFork() {
		return "", fmt.Errorf("cannot fork: no active Claude session")
	}

	workDir := i.ProjectPath
	configDir := GetClaudeConfigDir()

	// Capture-resume pattern for fork:
	// 1. Fork in print mode to get new session ID
	// 2. Store in tmux environment
	// 3. Resume the forked session interactively
	cmd := fmt.Sprintf(
		`cd %s && session_id=$(CLAUDE_CONFIG_DIR=%s claude -p "." --output-format json --resume %s --fork-session 2>/dev/null | jq -r '.session_id') && `+
			`tmux set-environment CLAUDE_SESSION_ID "$session_id" && `+
			`CLAUDE_CONFIG_DIR=%s claude --resume "$session_id" --dangerously-skip-permissions`,
		workDir, configDir, i.ClaudeSessionID, configDir)

	return cmd, nil
}

// GetActualWorkDir returns the actual working directory from tmux, or falls back to ProjectPath
func (i *Instance) GetActualWorkDir() string {
	if i.tmuxSession != nil {
		if workDir := i.tmuxSession.GetWorkDir(); workDir != "" {
			return workDir
		}
	}
	return i.ProjectPath
}

// CreateForkedInstance creates a new Instance configured for forking
func (i *Instance) CreateForkedInstance(newTitle, newGroupPath string) (*Instance, string, error) {
	cmd, err := i.Fork(newTitle, newGroupPath)
	if err != nil {
		return nil, "", err
	}

	// Create new instance with the PARENT's project path
	// This ensures the forked session is in the same Claude project directory as parent
	forked := NewInstance(newTitle, i.ProjectPath)
	if newGroupPath != "" {
		forked.GroupPath = newGroupPath
	} else {
		forked.GroupPath = i.GroupPath
	}
	forked.Command = cmd
	forked.Tool = "claude"

	return forked, cmd, nil
}

// Exists checks if the tmux session still exists
func (i *Instance) Exists() bool {
	if i.tmuxSession == nil {
		return false
	}
	return i.tmuxSession.Exists()
}

// GetTmuxSession returns the tmux session object
func (i *Instance) GetTmuxSession() *tmux.Session {
	return i.tmuxSession
}

// GetSessionIDFromTmux reads Claude session ID from tmux environment
// This is the primary method for sessions started with the capture-resume pattern
func (i *Instance) GetSessionIDFromTmux() string {
	if i.tmuxSession == nil {
		return ""
	}
	sessionID, err := i.tmuxSession.GetEnvironment("CLAUDE_SESSION_ID")
	if err != nil {
		return ""
	}
	return sessionID
}

// GetMCPInfo returns MCP server information for this session
// Returns nil if not a Claude session
func (i *Instance) GetMCPInfo() *MCPInfo {
	if i.Tool != "claude" {
		return nil
	}
	return GetMCPInfo(i.ProjectPath)
}

// CaptureLoadedMCPs captures the current MCP names as the "loaded" state
// This should be called when a session starts or restarts, so we can track
// which MCPs are actually loaded in the running Claude session vs just configured
func (i *Instance) CaptureLoadedMCPs() {
	if i.Tool != "claude" {
		i.LoadedMCPNames = nil
		return
	}

	mcpInfo := GetMCPInfo(i.ProjectPath)
	if mcpInfo == nil {
		i.LoadedMCPNames = nil
		return
	}

	i.LoadedMCPNames = mcpInfo.AllNames()
}

// generateID generates a unique session ID
func generateID() string {
	return fmt.Sprintf("%s-%d", randomString(8), time.Now().Unix())
}

// randomString generates a random hex string of specified length
func randomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// UpdateClaudeSessionsWithDedup updates Claude sessions for all instances with deduplication
// This should be called from the manager/storage layer that has access to all instances
// It both fixes existing duplicates AND prevents new duplicates during detection
func UpdateClaudeSessionsWithDedup(instances []*Instance) {
	// Sort instances by CreatedAt (older first get priority for keeping IDs)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].CreatedAt.Before(instances[j].CreatedAt)
	})

	// Step 1: Find and clear duplicate IDs (keep only the oldest session's claim)
	// Map from session ID to the instance that owns it (oldest one)
	idOwner := make(map[string]*Instance)
	for _, inst := range instances {
		if inst.Tool != "claude" || inst.ClaudeSessionID == "" {
			continue
		}
		if owner, exists := idOwner[inst.ClaudeSessionID]; exists {
			// Duplicate found! The older session (owner) keeps the ID
			// Clear the newer session's ID so it can re-detect
			inst.ClaudeSessionID = ""
			inst.ClaudeDetectedAt = time.Time{}
			_ = owner // Older session keeps its ID
		} else {
			idOwner[inst.ClaudeSessionID] = inst
		}
	}

	// Step 2: Build usedIDs from remaining assigned IDs
	usedIDs := make(map[string]bool)
	for id := range idOwner {
		usedIDs[id] = true
	}

	// Step 3: Re-detect for sessions that need it (empty or cleared IDs)
	for _, inst := range instances {
		if inst.Tool == "claude" && inst.ClaudeSessionID == "" {
			inst.UpdateClaudeSession(usedIDs)
			// If we found one, add to used IDs
			if inst.ClaudeSessionID != "" {
				usedIDs[inst.ClaudeSessionID] = true
			}
		}
	}
}
