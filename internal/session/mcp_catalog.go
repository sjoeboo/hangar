package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MCPServerConfig represents an MCP server configuration (Claude's format)
type MCPServerConfig struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// WriteMCPJsonFromConfig writes enabled MCPs from config.toml to project's .mcp.json
func WriteMCPJsonFromConfig(projectPath string, enabledNames []string) error {
	mcpFile := filepath.Join(projectPath, ".mcp.json")
	availableMCPs := GetAvailableMCPs()

	// Build the .mcp.json content using MCPServerConfig format (Claude's expected format)
	mcpConfig := struct {
		MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	}{
		MCPServers: make(map[string]MCPServerConfig),
	}

	for _, name := range enabledNames {
		if def, ok := availableMCPs[name]; ok {
			// Initialize empty slices/maps to avoid null in JSON output
			args := def.Args
			if args == nil {
				args = []string{}
			}
			env := def.Env
			if env == nil {
				env = map[string]string{}
			}
			mcpConfig.MCPServers[name] = MCPServerConfig{
				Type:    "stdio", // Default to stdio for npx/command-based MCPs
				Command: def.Command,
				Args:    args,
				Env:     env,
			}
		}
	}

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

	// Atomic write
	tmpPath := mcpFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .mcp.json: %w", err)
	}

	if err := os.Rename(tmpPath, mcpFile); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save .mcp.json: %w", err)
	}

	return nil
}

// WriteGlobalMCP adds or removes MCPs from Claude's global config
// This modifies ~/.claude-work/.claude.json → mcpServers
func WriteGlobalMCP(enabledNames []string) error {
	configDir := GetClaudeConfigDir()
	configFile := filepath.Join(configDir, ".claude.json")

	// Read existing config (preserve other fields like projects, settings, etc.)
	var rawConfig map[string]interface{}
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &rawConfig); err != nil {
			rawConfig = make(map[string]interface{})
		}
	} else {
		rawConfig = make(map[string]interface{})
	}

	// Build new mcpServers from enabled names using config.toml definitions
	availableMCPs := GetAvailableMCPs()
	mcpServers := make(map[string]MCPServerConfig)

	for _, name := range enabledNames {
		if def, ok := availableMCPs[name]; ok {
			args := def.Args
			if args == nil {
				args = []string{}
			}
			env := def.Env
			if env == nil {
				env = map[string]string{}
			}
			mcpServers[name] = MCPServerConfig{
				Type:    "stdio",
				Command: def.Command,
				Args:    args,
				Env:     env,
			}
		}
	}

	rawConfig["mcpServers"] = mcpServers

	// Write atomically
	data, err := json.MarshalIndent(rawConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpPath := configFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpPath, configFile); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetGlobalMCPNames returns the names of MCPs currently in Claude's global config
func GetGlobalMCPNames() []string {
	configDir := GetClaudeConfigDir()
	configFile := filepath.Join(configDir, ".claude.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}

	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	names := make([]string, 0, len(config.MCPServers))
	for name := range config.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetProjectMCPNames returns MCPs from projects[path].mcpServers in Claude's config
func GetProjectMCPNames(projectPath string) []string {
	configDir := GetClaudeConfigDir()
	configFile := filepath.Join(configDir, ".claude.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}

	var config struct {
		Projects map[string]struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	proj, ok := config.Projects[projectPath]
	if !ok {
		return nil
	}

	names := make([]string, 0, len(proj.MCPServers))
	for name := range proj.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ClearProjectMCPs removes all MCPs from projects[path].mcpServers in Claude's config
func ClearProjectMCPs(projectPath string) error {
	configDir := GetClaudeConfigDir()
	configFile := filepath.Join(configDir, ".claude.json")

	// Read existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get projects map
	projects, ok := rawConfig["projects"].(map[string]interface{})
	if !ok {
		return nil // No projects, nothing to clear
	}

	// Get specific project
	proj, ok := projects[projectPath].(map[string]interface{})
	if !ok {
		return nil // Project not found, nothing to clear
	}

	// Clear mcpServers for this project
	proj["mcpServers"] = map[string]interface{}{}

	// Write atomically
	newData, err := json.MarshalIndent(rawConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpPath := configFile + ".tmp"
	if err := os.WriteFile(tmpPath, newData, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpPath, configFile); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
