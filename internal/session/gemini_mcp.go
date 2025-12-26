package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// GeminiMCPConfig represents settings.json structure
// VERIFIED: Actual settings.json does NOT have mcp.allowed/excluded
// (Simplified structure compared to research docs)
type GeminiMCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	// Note: No MCP global settings in actual Gemini settings.json
}

// GetGeminiMCPInfo reads MCP configuration from settings.json
// Returns MCPInfo with Global MCPs only (Gemini has no project-level MCPs)
// VERIFIED: settings.json structure is simple {"mcpServers": {...}}
func GetGeminiMCPInfo(projectPath string) *MCPInfo {
	configFile := filepath.Join(GetGeminiConfigDir(), "settings.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		return &MCPInfo{}
	}

	var config GeminiMCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &MCPInfo{}
	}

	info := &MCPInfo{}

	// All MCPs are global in Gemini (no project or local MCPs)
	for name := range config.MCPServers {
		info.Global = append(info.Global, name)
	}

	sort.Strings(info.Global)
	return info
}

// contains checks if a slice contains a string
// Helper function for tests and internal use
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
