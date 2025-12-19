package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ClaudeProject represents a project entry in Claude's config
type ClaudeProject struct {
	LastSessionId string `json:"lastSessionId"`
}

// ClaudeConfig represents the structure of .claude.json
type ClaudeConfig struct {
	Projects map[string]ClaudeProject `json:"projects"`
}

// MCPInfo contains MCP server information for a session
type MCPInfo struct {
	Global  []string // From CLAUDE_CONFIG_DIR/.claude.json mcpServers
	Project []string // From CLAUDE_CONFIG_DIR/.claude.json projects[path].mcpServers
	Local   []string // From {projectPath}/.mcp.json mcpServers
}

// HasAny returns true if any MCPs are configured
func (m *MCPInfo) HasAny() bool {
	return len(m.Global) > 0 || len(m.Project) > 0 || len(m.Local) > 0
}

// Total returns total number of MCPs across all sources
func (m *MCPInfo) Total() int {
	return len(m.Global) + len(m.Project) + len(m.Local)
}

// claudeConfigForMCP is used for parsing MCP-related fields from .claude.json
type claudeConfigForMCP struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
	Projects   map[string]struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	} `json:"projects"`
}

// projectMCPConfig is used for parsing .mcp.json files
type projectMCPConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// MCP info cache (30 second TTL to avoid re-reading files on every render)
var (
	mcpInfoCache   = make(map[string]*MCPInfo)
	mcpInfoCacheMu sync.RWMutex
	mcpCacheExpiry = 30 * time.Second
	mcpCacheTimes  = make(map[string]time.Time)
)

// MCPServer represents an MCP with its enabled state
type MCPServer struct {
	Name    string
	Source  string // "local", "global", "project"
	Enabled bool
}

// ProjectMCPSettings represents .claude/settings.local.json
type ProjectMCPSettings struct {
	EnableAllProjectMcpServers bool     `json:"enableAllProjectMcpServers,omitempty"`
	EnabledMcpjsonServers      []string `json:"enabledMcpjsonServers,omitempty"`
	DisabledMcpjsonServers     []string `json:"disabledMcpjsonServers,omitempty"`
}

// MCPMode indicates how MCP enabling/disabling is configured
type MCPMode int

const (
	MCPModeDefault   MCPMode = iota // No explicit config, all enabled
	MCPModeWhitelist                // enabledMcpjsonServers is set
	MCPModeBlacklist                // disabledMcpjsonServers is set
)

// GetMCPInfo retrieves MCP server information for a project path (cached)
// It reads from three sources:
// 1. Global MCPs: CLAUDE_CONFIG_DIR/.claude.json → mcpServers
// 2. Project MCPs: CLAUDE_CONFIG_DIR/.claude.json → projects[projectPath].mcpServers
// 3. Local MCPs: {projectPath}/.mcp.json → mcpServers
func GetMCPInfo(projectPath string) *MCPInfo {
	// Check cache first
	mcpInfoCacheMu.RLock()
	if cached, ok := mcpInfoCache[projectPath]; ok {
		if time.Since(mcpCacheTimes[projectPath]) < mcpCacheExpiry {
			mcpInfoCacheMu.RUnlock()
			return cached
		}
	}
	mcpInfoCacheMu.RUnlock()

	// Cache miss or expired - fetch fresh data
	info := getMCPInfoUncached(projectPath)

	// Update cache
	mcpInfoCacheMu.Lock()
	mcpInfoCache[projectPath] = info
	mcpCacheTimes[projectPath] = time.Now()
	mcpInfoCacheMu.Unlock()

	return info
}

// getMCPInfoUncached reads MCP info from disk (called by cached wrapper)
func getMCPInfoUncached(projectPath string) *MCPInfo {
	info := &MCPInfo{}
	configDir := GetClaudeConfigDir()

	// Read .claude.json for global and project MCPs
	configFile := filepath.Join(configDir, ".claude.json")
	if data, err := os.ReadFile(configFile); err == nil {
		var config claudeConfigForMCP
		if json.Unmarshal(data, &config) == nil {
			// Global MCPs
			for name := range config.MCPServers {
				info.Global = append(info.Global, name)
			}
			// Project-specific MCPs
			if proj, ok := config.Projects[projectPath]; ok {
				for name := range proj.MCPServers {
					info.Project = append(info.Project, name)
				}
			}
		}
	}

	// Read .mcp.json from project directory for local MCPs
	mcpFile := filepath.Join(projectPath, ".mcp.json")
	if data, err := os.ReadFile(mcpFile); err == nil {
		var mcp projectMCPConfig
		if json.Unmarshal(data, &mcp) == nil {
			for name := range mcp.MCPServers {
				info.Local = append(info.Local, name)
			}
		}
	}

	// Sort for consistent display
	sort.Strings(info.Global)
	sort.Strings(info.Project)
	sort.Strings(info.Local)

	return info
}

// GetClaudeConfigDir returns the Claude config directory
// Priority: 1) CLAUDE_CONFIG_DIR env, 2) UserConfig setting, 3) ~/.claude
func GetClaudeConfigDir() string {
	// 1. Check env var (highest priority)
	if envDir := os.Getenv("CLAUDE_CONFIG_DIR"); envDir != "" {
		return expandTilde(envDir)
	}

	// 2. Check user config
	userConfig, _ := LoadUserConfig()
	if userConfig != nil && userConfig.Claude.ConfigDir != "" {
		return expandTilde(userConfig.Claude.ConfigDir)
	}

	// 3. Default to ~/.claude
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// GetClaudeSessionID returns the ACTIVE session ID for a project path
// It first tries to find the currently running session by checking recently
// modified .jsonl files, then falls back to lastSessionId from config
func GetClaudeSessionID(projectPath string) (string, error) {
	configDir := GetClaudeConfigDir()

	// First, try to find active session from recently modified files
	activeID := findActiveSessionID(configDir, projectPath)
	if activeID != "" {
		return activeID, nil
	}

	// Fall back to lastSessionId from config
	configFile := filepath.Join(configDir, ".claude.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to read Claude config: %w", err)
	}

	var config ClaudeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse Claude config: %w", err)
	}

	// Look up project by path
	if project, ok := config.Projects[projectPath]; ok {
		if project.LastSessionId != "" {
			return project.LastSessionId, nil
		}
	}

	return "", fmt.Errorf("no session found for project: %s", projectPath)
}

// findActiveSessionID looks for the most recently modified session file
// This finds the CURRENTLY RUNNING session, not the last completed one
func findActiveSessionID(configDir, projectPath string) string {
	// Convert project path to Claude's directory format
	// /Users/ashesh/claude-deck -> -Users-ashesh-claude-deck
	projectDirName := strings.ReplaceAll(projectPath, "/", "-")
	projectDir := filepath.Join(configDir, "projects", projectDirName)

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return ""
	}

	// Find session files (UUID format, not agent-* files)
	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil || len(files) == 0 {
		return ""
	}

	// UUID pattern for session files
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jsonl$`)

	var mostRecent string
	var mostRecentTime time.Time

	for _, file := range files {
		base := filepath.Base(file)

		// Skip agent files (agent-*.jsonl)
		if strings.HasPrefix(base, "agent-") {
			continue
		}

		// Only consider UUID-named files
		if !uuidPattern.MatchString(base) {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Find the most recently modified file
		if info.ModTime().After(mostRecentTime) {
			mostRecentTime = info.ModTime()
			mostRecent = strings.TrimSuffix(base, ".jsonl")
		}
	}

	// Only return if modified within last 5 minutes (actively used)
	if mostRecent != "" && time.Since(mostRecentTime) < 5*time.Minute {
		return mostRecent
	}

	return ""
}

// FindSessionForInstance finds the session file for a specific instance
// Parameters:
//   - projectPath: the project directory
//   - createdAfter: only consider files with internal timestamp >= this time
//   - excludeIDs: session IDs already claimed by other instances
//
// Returns the session ID or empty string if not found
//
// Note: This function uses internal timestamp as the primary filter.
// For FORKED sessions (where internal timestamp is inherited from parent),
// we also check if the file was CREATED very recently (ModTime within 30s of now),
// which indicates it's a newly created fork file, not an actively updated conversation.
func FindSessionForInstance(projectPath string, createdAfter time.Time, excludeIDs map[string]bool) string {
	configDir := GetClaudeConfigDir()

	// Convert project path to Claude's directory format
	// /Users/ashesh/claude-deck -> -Users-ashesh-claude-deck
	projectDirName := strings.ReplaceAll(projectPath, "/", "-")
	projectDir := filepath.Join(configDir, "projects", projectDirName)

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return ""
	}

	// Find all UUID-named session files
	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil || len(files) == 0 {
		return ""
	}

	// UUID pattern for session files
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jsonl$`)

	type candidate struct {
		sessionID   string
		timestamp   time.Time // Internal timestamp (for sorting)
		fileModTime time.Time // File modification time
		isNewFork   bool      // True if this appears to be a newly created fork file
	}
	var candidates []candidate

	now := time.Now()

	for _, file := range files {
		base := filepath.Base(file)

		// Skip agent files
		if strings.HasPrefix(base, "agent-") {
			continue
		}

		// Only UUID-named files
		if !uuidPattern.MatchString(base) {
			continue
		}

		sessionID := strings.TrimSuffix(base, ".jsonl")

		// Skip if already claimed by another instance
		if excludeIDs[sessionID] {
			continue
		}

		// Get file info for ModTime
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileModTime := info.ModTime()

		// Get internal timestamp from file
		internalTS := getFileInternalTimestamp(file)

		// Primary check: Internal timestamp is after createdAfter (works for new sessions)
		internalTSValid := !internalTS.IsZero() && !internalTS.Before(createdAfter)

		// Secondary check for FORKS: File was created very recently (within 30s of now)
		// AND the file ModTime is after our instance creation time.
		// This catches fork files where internal timestamp is inherited from parent.
		// We use a tight 30s window to avoid matching actively-used conversation files.
		isNewFork := false
		if !internalTSValid && !fileModTime.Before(createdAfter) {
			// File was modified after our session started - could be fork OR active conversation
			// Only consider it a fork if ModTime is VERY recent (file just created)
			if now.Sub(fileModTime) < 30*time.Second {
				isNewFork = true
			}
		}

		if !internalTSValid && !isNewFork {
			continue
		}

		// Use internal timestamp for sorting if valid, otherwise use file ModTime
		sortTime := internalTS
		if internalTS.IsZero() || isNewFork {
			sortTime = fileModTime
		}

		candidates = append(candidates, candidate{
			sessionID:   sessionID,
			timestamp:   sortTime,
			fileModTime: fileModTime,
			isNewFork:   isNewFork,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates: prefer new sessions (internal timestamp valid) over forks
	// Within each category, sort by timestamp (earliest first)
	sort.Slice(candidates, func(i, j int) bool {
		// Prefer non-fork candidates (internal timestamp matched)
		if !candidates[i].isNewFork && candidates[j].isNewFork {
			return true
		}
		if candidates[i].isNewFork && !candidates[j].isNewFork {
			return false
		}
		// Within same category, sort by timestamp
		return candidates[i].timestamp.Before(candidates[j].timestamp)
	})

	return candidates[0].sessionID
}

// getFileInternalTimestamp reads a session file and extracts the earliest timestamp
// It scans up to 10 lines because some files start with summary lines that have no timestamp
func getFileInternalTimestamp(filePath string) time.Time {
	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Scan up to 10 lines looking for a timestamp
	for i := 0; i < 10 && scanner.Scan(); i++ {
		line := scanner.Text()

		// Parse JSON to get timestamp field (direct or in snapshot)
		var data struct {
			Timestamp string `json:"timestamp"`
			Snapshot  struct {
				Timestamp string `json:"timestamp"`
			} `json:"snapshot"`
		}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		// Try direct timestamp first, then snapshot.timestamp
		tsStr := data.Timestamp
		if tsStr == "" {
			tsStr = data.Snapshot.Timestamp
		}
		if tsStr == "" {
			continue
		}

		// Parse ISO 8601 timestamp
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			// Try parsing with milliseconds
			ts, err = time.Parse("2006-01-02T15:04:05.999Z", tsStr)
			if err != nil {
				continue
			}
		}

		return ts
	}

	return time.Time{}
}

// getProjectSettingsPath returns the path to .claude/settings.local.json for a project
func getProjectSettingsPath(projectPath string) string {
	return filepath.Join(projectPath, ".claude", "settings.local.json")
}

// readProjectMCPSettings reads the project's MCP settings file
func readProjectMCPSettings(projectPath string) (*ProjectMCPSettings, error) {
	settingsPath := getProjectSettingsPath(projectPath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No settings file = default (all enabled)
			return &ProjectMCPSettings{}, nil
		}
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var settings ProjectMCPSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}

	return &settings, nil
}

// GetMCPMode determines the MCP configuration mode for a project
func GetMCPMode(projectPath string) MCPMode {
	settings, err := readProjectMCPSettings(projectPath)
	if err != nil {
		return MCPModeDefault
	}

	// Whitelist takes priority if set
	if len(settings.EnabledMcpjsonServers) > 0 {
		return MCPModeWhitelist
	}

	// Check for blacklist
	if len(settings.DisabledMcpjsonServers) > 0 {
		return MCPModeBlacklist
	}

	return MCPModeDefault
}

// GetLocalMCPState returns Local MCPs with their enabled state
func GetLocalMCPState(projectPath string) ([]MCPServer, error) {
	// Get all Local MCPs from .mcp.json
	mcpFile := filepath.Join(projectPath, ".mcp.json")
	data, err := os.ReadFile(mcpFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No .mcp.json = no Local MCPs
		}
		return nil, fmt.Errorf("failed to read .mcp.json: %w", err)
	}

	var mcpConfig projectMCPConfig
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		return nil, fmt.Errorf("failed to parse .mcp.json: %w", err)
	}

	if len(mcpConfig.MCPServers) == 0 {
		return nil, nil
	}

	// Get settings to determine enabled state
	settings, err := readProjectMCPSettings(projectPath)
	if err != nil {
		return nil, err
	}

	mode := GetMCPMode(projectPath)

	// Build result with enabled state
	var servers []MCPServer
	for name := range mcpConfig.MCPServers {
		enabled := isMCPEnabled(name, settings, mode)
		servers = append(servers, MCPServer{
			Name:    name,
			Source:  "local",
			Enabled: enabled,
		})
	}

	// Sort for consistent display
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}

// isMCPEnabled determines if an MCP is enabled based on settings and mode
func isMCPEnabled(name string, settings *ProjectMCPSettings, mode MCPMode) bool {
	switch mode {
	case MCPModeWhitelist:
		// Whitelist: enabled only if in enabledMcpjsonServers
		for _, enabled := range settings.EnabledMcpjsonServers {
			if enabled == name {
				return true
			}
		}
		return false

	case MCPModeBlacklist:
		// Blacklist: enabled unless in disabledMcpjsonServers
		for _, disabled := range settings.DisabledMcpjsonServers {
			if disabled == name {
				return false
			}
		}
		return true

	default:
		// Default: all enabled
		return true
	}
}

// ClearMCPCache invalidates the MCP cache for a project path
func ClearMCPCache(projectPath string) {
	mcpInfoCacheMu.Lock()
	delete(mcpInfoCache, projectPath)
	delete(mcpCacheTimes, projectPath)
	mcpInfoCacheMu.Unlock()
}

// ToggleLocalMCP toggles a Local MCP on/off
// It respects the existing mode (whitelist vs blacklist) or initializes with blacklist
func ToggleLocalMCP(projectPath, mcpName string) error {
	// Read current settings (preserving other fields)
	settingsPath := getProjectSettingsPath(projectPath)
	var rawSettings map[string]interface{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read settings: %w", err)
		}
		// File doesn't exist, start fresh
		rawSettings = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &rawSettings); err != nil {
			return fmt.Errorf("failed to parse settings: %w", err)
		}
	}

	// Detect mode
	mode := GetMCPMode(projectPath)

	// Get current enabled state
	settings, _ := readProjectMCPSettings(projectPath)
	currentlyEnabled := isMCPEnabled(mcpName, settings, mode)

	// Toggle based on mode
	switch mode {
	case MCPModeWhitelist:
		// Modify enabledMcpjsonServers
		enabled := getStringSlice(rawSettings, "enabledMcpjsonServers")
		if currentlyEnabled {
			// Disable: remove from whitelist
			enabled = removeFromSlice(enabled, mcpName)
		} else {
			// Enable: add to whitelist
			enabled = appendIfMissing(enabled, mcpName)
		}
		rawSettings["enabledMcpjsonServers"] = enabled

	case MCPModeBlacklist:
		// Modify disabledMcpjsonServers
		disabled := getStringSlice(rawSettings, "disabledMcpjsonServers")
		if currentlyEnabled {
			// Disable: add to blacklist
			disabled = appendIfMissing(disabled, mcpName)
		} else {
			// Enable: remove from blacklist
			disabled = removeFromSlice(disabled, mcpName)
		}
		rawSettings["disabledMcpjsonServers"] = disabled

	default:
		// No mode set, initialize with blacklist
		if currentlyEnabled {
			// Disable: add to blacklist
			rawSettings["disabledMcpjsonServers"] = []string{mcpName}
		}
		// Enable does nothing (already enabled by default)
	}

	// Ensure .claude directory exists
	claudeDir := filepath.Join(projectPath, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Write atomically (temp file + rename)
	newData, err := json.MarshalIndent(rawSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	tmpPath := settingsPath + ".tmp"
	if err := os.WriteFile(tmpPath, newData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, settingsPath); err != nil {
		os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("failed to rename settings file: %w", err)
	}

	// Clear cache so changes are reflected
	ClearMCPCache(projectPath)

	return nil
}

// getStringSlice extracts a string slice from a map
func getStringSlice(m map[string]interface{}, key string) []string {
	val, ok := m[key]
	if !ok {
		return nil
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// removeFromSlice removes a string from a slice
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// appendIfMissing adds a string to a slice if not already present
func appendIfMissing(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
