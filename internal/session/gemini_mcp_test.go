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
