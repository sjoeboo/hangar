package session

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

// UserConfigFileName is the TOML config file for user preferences
const UserConfigFileName = "config.toml"

// UserConfig represents user-facing configuration in TOML format
type UserConfig struct {
	// Tools defines custom AI tool configurations
	Tools map[string]ToolDef `toml:"tools"`

	// Claude defines Claude Code integration settings
	Claude ClaudeSettings `toml:"claude"`
}

// ClaudeSettings defines Claude Code configuration
type ClaudeSettings struct {
	// ConfigDir is the path to Claude's config directory
	// Default: ~/.claude (or CLAUDE_CONFIG_DIR env var)
	ConfigDir string `toml:"config_dir"`
}

// ToolDef defines a custom AI tool
type ToolDef struct {
	// Command is the shell command to run
	Command string `toml:"command"`

	// Icon is the emoji/symbol to display
	Icon string `toml:"icon"`

	// BusyPatterns are strings that indicate the tool is busy
	BusyPatterns []string `toml:"busy_patterns"`
}

// Default user config (empty tools map)
var defaultUserConfig = UserConfig{
	Tools: make(map[string]ToolDef),
}

// Cache for user config (loaded once per session)
var (
	userConfigCache   *UserConfig
	userConfigCacheMu sync.RWMutex
)

// GetUserConfigPath returns the path to the user config file
func GetUserConfigPath() (string, error) {
	dir, err := GetAgentDeckDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, UserConfigFileName), nil
}

// LoadUserConfig loads the user configuration from TOML file
// Returns cached config after first load
func LoadUserConfig() (*UserConfig, error) {
	userConfigCacheMu.RLock()
	if userConfigCache != nil {
		defer userConfigCacheMu.RUnlock()
		return userConfigCache, nil
	}
	userConfigCacheMu.RUnlock()

	// Load config (only happens once)
	userConfigCacheMu.Lock()
	defer userConfigCacheMu.Unlock()

	// Double-check after acquiring write lock
	if userConfigCache != nil {
		return userConfigCache, nil
	}

	configPath, err := GetUserConfigPath()
	if err != nil {
		userConfigCache = &defaultUserConfig
		return userConfigCache, nil
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config (no file exists yet)
		userConfigCache = &defaultUserConfig
		return userConfigCache, nil
	}

	var config UserConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		// Return default on parse error (don't crash)
		userConfigCache = &defaultUserConfig
		return userConfigCache, nil
	}

	// Initialize maps if nil
	if config.Tools == nil {
		config.Tools = make(map[string]ToolDef)
	}

	userConfigCache = &config
	return userConfigCache, nil
}

// ReloadUserConfig forces a reload of the user config
func ReloadUserConfig() (*UserConfig, error) {
	userConfigCacheMu.Lock()
	userConfigCache = nil
	userConfigCacheMu.Unlock()
	return LoadUserConfig()
}

// GetToolDef returns a tool definition from user config
// Returns nil if tool is not defined
func GetToolDef(toolName string) *ToolDef {
	config, err := LoadUserConfig()
	if err != nil || config == nil {
		return nil
	}

	if def, ok := config.Tools[toolName]; ok {
		return &def
	}
	return nil
}

// GetToolIcon returns the icon for a tool (custom or built-in)
func GetToolIcon(toolName string) string {
	// Check custom tools first
	if def := GetToolDef(toolName); def != nil && def.Icon != "" {
		return def.Icon
	}

	// Built-in icons
	switch toolName {
	case "claude":
		return "🤖"
	case "gemini":
		return "✨"
	case "aider":
		return "🔧"
	case "codex":
		return "💻"
	case "cursor":
		return "📝"
	case "shell":
		return "🐚"
	default:
		return "🐚"
	}
}

// GetToolBusyPatterns returns busy patterns for a tool (custom + built-in)
func GetToolBusyPatterns(toolName string) []string {
	var patterns []string

	// Add custom patterns first
	if def := GetToolDef(toolName); def != nil {
		patterns = append(patterns, def.BusyPatterns...)
	}

	// Built-in patterns are handled by the detector
	return patterns
}

// CreateExampleConfig creates an example config file if none exists
func CreateExampleConfig() error {
	configPath, err := GetUserConfigPath()
	if err != nil {
		return err
	}

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	exampleConfig := `# Agent Deck User Configuration
# This file is loaded on startup. Edit to customize tools.

# Claude Code integration
# Set this if you use a custom Claude profile (e.g., dual account setup)
# Default: ~/.claude (or CLAUDE_CONFIG_DIR env var takes priority)
# [claude]
# config_dir = "~/.claude-work"

# Custom tool definitions
# Each tool can have:
#   command      - The shell command to run
#   icon         - Emoji/symbol shown in the UI
#   busy_patterns - Strings that indicate the tool is processing

# Example: Add a custom AI tool
# [tools.my-ai]
# command = "my-ai-assistant"
# icon = "🧠"
# busy_patterns = ["thinking...", "processing..."]

# Example: Add GitHub Copilot CLI
# [tools.copilot]
# command = "gh copilot"
# icon = "🤖"
# busy_patterns = ["Generating..."]
`

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(configPath, []byte(exampleConfig), 0600)
}
