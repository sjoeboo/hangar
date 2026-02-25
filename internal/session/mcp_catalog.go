package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"ghe.spotify.net/mnicholson/hangar/internal/logging"
)

var mcpCatLog = logging.ForComponent(logging.CompMCP)

// MCPServerConfig represents an MCP server configuration (Claude's format)
type MCPServerConfig struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`     // For HTTP transport
	Headers map[string]string `json:"headers,omitempty"` // For HTTP transport (e.g., Authorization)
}

// readExistingLocalMCPServers reads mcpServers from an existing .mcp.json file.
// Returns nil if the file doesn't exist or can't be parsed.
func readExistingLocalMCPServers(mcpFile string) map[string]json.RawMessage {
	data, err := os.ReadFile(mcpFile)
	if err != nil {
		return nil
	}
	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}
	return config.MCPServers
}

// WriteMCPJsonFromConfig writes enabled MCPs from config.toml to project's .mcp.json
// It preserves any existing entries not managed by hangar (not defined in config.toml)
func WriteMCPJsonFromConfig(projectPath string, enabledNames []string) error {
	if !GetManageMCPJson() {
		mcpCatLog.Debug("mcp_json_management_disabled", slog.String("path", projectPath))
		return nil
	}

	mcpFile := filepath.Join(projectPath, ".mcp.json")
	availableMCPs := GetAvailableMCPs()

	// Read existing .mcp.json to preserve entries not managed by hangar (#146)
	existingServers := readExistingLocalMCPServers(mcpFile)

	// Build hangar managed MCP entries
	hangarServers := make(map[string]MCPServerConfig)

	for _, name := range enabledNames {
		if def, ok := availableMCPs[name]; ok {
			// Check if this is an HTTP/SSE MCP (has URL configured)
			if def.URL != "" {
				transport := def.Transport
				if transport == "" {
					transport = "http"
				}
				hangarServers[name] = MCPServerConfig{
					Type:    transport,
					URL:     def.URL,
					Headers: def.Headers,
				}
				mcpCatLog.Info("transport_http", slog.String("mcp", name), slog.String("scope", "local"), slog.String("transport", transport), slog.String("url", def.URL))
				continue
			}

			// Use stdio mode
			args := def.Args
			if args == nil {
				args = []string{}
			}
			env := def.Env
			if env == nil {
				env = map[string]string{}
			}
			hangarServers[name] = MCPServerConfig{
				Type:    "stdio",
				Command: def.Command,
				Args:    args,
				Env:     env,
			}
			mcpCatLog.Info("transport_stdio", slog.String("mcp", name), slog.String("scope", "local"))
		}
	}

	// Merge: preserve non-hangar entries, then add hangar entries (#146)
	mergedServers := make(map[string]json.RawMessage)
	for name, raw := range existingServers {
		if _, managed := availableMCPs[name]; !managed {
			mergedServers[name] = raw
			mcpCatLog.Debug("preserved_existing_mcp", slog.String("mcp", name), slog.String("scope", "local"))
		}
	}
	for name, cfg := range hangarServers {
		raw, err := json.Marshal(cfg)
		if err != nil {
			mcpCatLog.Warn("marshal_mcp_entry_failed", slog.String("mcp", name), slog.Any("error", err))
			continue
		}
		mergedServers[name] = raw
	}

	finalConfig := struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}{
		MCPServers: mergedServers,
	}

	data, err := json.MarshalIndent(finalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

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
func WriteGlobalMCP(enabledNames []string) error {
	configDir := GetClaudeConfigDir()
	configFile := filepath.Join(configDir, ".claude.json")

	var rawConfig map[string]interface{}
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &rawConfig); err != nil {
			rawConfig = make(map[string]interface{})
		}
	} else {
		rawConfig = make(map[string]interface{})
	}

	availableMCPs := GetAvailableMCPs()
	mcpServers := make(map[string]MCPServerConfig)

	for _, name := range enabledNames {
		if def, ok := availableMCPs[name]; ok {
			if def.URL != "" {
				transport := def.Transport
				if transport == "" {
					transport = "http"
				}
				mcpServers[name] = MCPServerConfig{
					Type:    transport,
					URL:     def.URL,
					Headers: def.Headers,
				}
				mcpCatLog.Info("transport_http", slog.String("mcp", name), slog.String("scope", "global"), slog.String("transport", transport), slog.String("url", def.URL))
				continue
			}

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
			mcpCatLog.Info("transport_stdio", slog.String("mcp", name), slog.String("scope", "global"))
		}
	}

	mergedMCPs := make(map[string]interface{})
	if existingMCPs, ok := rawConfig["mcpServers"].(map[string]interface{}); ok {
		for name, cfg := range existingMCPs {
			if _, managed := availableMCPs[name]; !managed {
				mergedMCPs[name] = cfg
			}
		}
	}
	for name, cfg := range mcpServers {
		mergedMCPs[name] = cfg
	}
	rawConfig["mcpServers"] = mergedMCPs

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

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	projects, ok := rawConfig["projects"].(map[string]interface{})
	if !ok {
		return nil
	}

	proj, ok := projects[projectPath].(map[string]interface{})
	if !ok {
		return nil
	}

	proj["mcpServers"] = map[string]interface{}{}

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

// GetUserMCPRootPath returns the path to ~/.claude.json (ROOT config, always read by Claude)
func GetUserMCPRootPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude.json")
}

// WriteUserMCP writes MCPs to ~/.claude.json (ROOT config)
// WARNING: MCPs written here affect ALL Claude sessions regardless of profile!
func WriteUserMCP(enabledNames []string) error {
	configFile := GetUserMCPRootPath()
	if configFile == "" {
		return fmt.Errorf("could not determine home directory")
	}

	var rawConfig map[string]interface{}
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &rawConfig); err != nil {
			rawConfig = make(map[string]interface{})
		}
	} else {
		rawConfig = make(map[string]interface{})
	}

	availableMCPs := GetAvailableMCPs()
	mcpServers := make(map[string]MCPServerConfig)

	for _, name := range enabledNames {
		if def, ok := availableMCPs[name]; ok {
			if def.URL != "" {
				transport := def.Transport
				if transport == "" {
					transport = "http"
				}
				mcpServers[name] = MCPServerConfig{
					Type:    transport,
					URL:     def.URL,
					Headers: def.Headers,
				}
				mcpCatLog.Info("transport_http", slog.String("mcp", name), slog.String("scope", "user"), slog.String("transport", transport), slog.String("url", def.URL))
				continue
			}

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
			mcpCatLog.Info("transport_stdio", slog.String("mcp", name), slog.String("scope", "user"))
		}
	}

	mergedMCPs := make(map[string]interface{})
	if existingMCPs, ok := rawConfig["mcpServers"].(map[string]interface{}); ok {
		for name, cfg := range existingMCPs {
			if _, managed := availableMCPs[name]; !managed {
				mergedMCPs[name] = cfg
			}
		}
	}
	for name, cfg := range mcpServers {
		mergedMCPs[name] = cfg
	}
	rawConfig["mcpServers"] = mergedMCPs

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

// GetUserMCPNames returns the names of MCPs in ~/.claude.json (ROOT config)
func GetUserMCPNames() []string {
	configFile := GetUserMCPRootPath()
	if configFile == "" {
		return nil
	}

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
