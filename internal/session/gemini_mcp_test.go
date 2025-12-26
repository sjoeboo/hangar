package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseGeminiSettings(t *testing.T) {
	// VERIFIED: Actual settings.json structure (simplified)
	settingsJSON := `{
  "security": {
    "auth": {
      "selectedType": "oauth-personal"
    }
  },
  "mcpServers": {
    "exa": {
      "command": "npx",
      "args": ["-y", "exa-mcp-server"],
      "env": {"EXA_API_KEY": "$EXA_API_KEY"}
    },
    "firecrawl": {
      "command": "npx",
      "args": ["-y", "@mendable/firecrawl-mcp"]
    }
  }
}`

	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")
	os.WriteFile(settingsFile, []byte(settingsJSON), 0644)

	var config GeminiMCPConfig
	data, _ := os.ReadFile(settingsFile)
	err := json.Unmarshal(data, &config)

	if err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 MCP servers, got %d", len(config.MCPServers))
	}

	// VERIFIED: No mcp.allowed/excluded in actual Gemini settings.json
	// (Our research was wrong - actual file doesn't have this)
}

func TestGetGeminiMCPInfo(t *testing.T) {
	tmpDir := t.TempDir()
	// geminiConfigDirOverride replaces the full ~/.gemini path
	geminiConfigDirOverride = tmpDir
	defer func() { geminiConfigDirOverride = "" }()

	// Settings file is directly under the config dir
	settingsFile := filepath.Join(tmpDir, "settings.json")

	settingsJSON := `{
  "mcpServers": {
    "exa": {"command": "npx", "args": ["-y", "exa-mcp-server"]},
    "firecrawl": {"command": "npx", "args": ["-y", "@mendable/firecrawl-mcp"]}
  }
}`
	os.WriteFile(settingsFile, []byte(settingsJSON), 0644)

	info := GetGeminiMCPInfo("/any/path")

	if len(info.Global) != 2 {
		t.Errorf("Expected 2 global MCPs, got %d: %v", len(info.Global), info.Global)
	}

	if !contains(info.Global, "exa") || !contains(info.Global, "firecrawl") {
		t.Errorf("Expected exa and firecrawl in Global, got %v", info.Global)
	}

	// Gemini has no project or local MCPs
	if len(info.Project) != 0 {
		t.Error("Gemini should have no Project MCPs")
	}
	if len(info.LocalMCPs) != 0 {
		t.Error("Gemini should have no Local MCPs")
	}
}

func TestGetGeminiMCPInfo_NoSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	geminiConfigDirOverride = tmpDir
	defer func() { geminiConfigDirOverride = "" }()

	// Don't create settings.json

	info := GetGeminiMCPInfo("/any/path")

	// Should return empty MCPInfo, not nil
	if info == nil {
		t.Fatal("GetGeminiMCPInfo should return empty MCPInfo, not nil")
	}
	if len(info.Global) != 0 {
		t.Errorf("Expected 0 global MCPs when no settings file, got %d", len(info.Global))
	}
}

func TestGetGeminiMCPInfo_EmptyMCPServers(t *testing.T) {
	tmpDir := t.TempDir()
	// geminiConfigDirOverride replaces the full ~/.gemini path
	geminiConfigDirOverride = tmpDir
	defer func() { geminiConfigDirOverride = "" }()

	// Settings file is directly under the config dir
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Settings file with no mcpServers
	settingsJSON := `{"security": {"auth": {"selectedType": "oauth"}}}`
	os.WriteFile(settingsFile, []byte(settingsJSON), 0644)

	info := GetGeminiMCPInfo("/any/path")

	if len(info.Global) != 0 {
		t.Errorf("Expected 0 global MCPs when mcpServers empty, got %d", len(info.Global))
	}
}
