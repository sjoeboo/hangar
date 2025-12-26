package session

// GeminiMCPConfig represents settings.json structure
// VERIFIED: Actual settings.json does NOT have mcp.allowed/excluded
// (Simplified structure compared to research docs)
type GeminiMCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	// Note: No MCP global settings in actual Gemini settings.json
}
