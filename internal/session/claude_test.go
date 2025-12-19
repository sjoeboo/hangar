package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetClaudeConfigDir_Default(t *testing.T) {
	// Unset env var to test default/config behavior
	os.Unsetenv("CLAUDE_CONFIG_DIR")

	dir := GetClaudeConfigDir()
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".claude")

	// If user config exists with claude.config_dir, that takes precedence
	// Otherwise, default to ~/.claude
	userConfig, _ := LoadUserConfig()
	if userConfig != nil && userConfig.Claude.ConfigDir != "" {
		// Config exists, just verify we get a valid path
		if dir == "" {
			t.Error("GetClaudeConfigDir() returned empty string")
		}
	} else {
		// No config, should return default
		if dir != defaultPath {
			t.Errorf("GetClaudeConfigDir() = %s, want %s", dir, defaultPath)
		}
	}
}

func TestGetClaudeConfigDir_EnvOverride(t *testing.T) {
	os.Setenv("CLAUDE_CONFIG_DIR", "/custom/path")
	defer os.Unsetenv("CLAUDE_CONFIG_DIR")

	dir := GetClaudeConfigDir()
	if dir != "/custom/path" {
		t.Errorf("GetClaudeConfigDir() = %s, want /custom/path", dir)
	}
}

func TestGetClaudeSessionID_NotFound(t *testing.T) {
	id, err := GetClaudeSessionID("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
	if id != "" {
		t.Errorf("Expected empty ID, got %s", id)
	}
}

func TestGetMCPInfo_Empty(t *testing.T) {
	// Use isolated config dir to avoid picking up real global MCPs
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", "/nonexistent/config/dir")
	defer func() {
		if oldConfigDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	// Test with non-existent path - should return empty MCPInfo, no panic
	info := GetMCPInfo("/nonexistent/path/that/does/not/exist")

	if info == nil {
		t.Fatal("GetMCPInfo returned nil for non-existent path")
	}
	if info.HasAny() {
		t.Error("Expected empty MCPInfo for non-existent path")
	}
	if info.Total() != 0 {
		t.Errorf("Expected Total()=0, got %d", info.Total())
	}
}

func TestMCPInfo_HasAny(t *testing.T) {
	tests := []struct {
		name     string
		info     MCPInfo
		expected bool
	}{
		{"empty", MCPInfo{}, false},
		{"global only", MCPInfo{Global: []string{"server1"}}, true},
		{"project only", MCPInfo{Project: []string{"server1"}}, true},
		{"local only", MCPInfo{Local: []string{"server1"}}, true},
		{"all", MCPInfo{Global: []string{"a"}, Project: []string{"b"}, Local: []string{"c"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.HasAny(); got != tt.expected {
				t.Errorf("HasAny() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMCPInfo_Total(t *testing.T) {
	info := MCPInfo{
		Global:  []string{"a", "b"},
		Project: []string{"c"},
		Local:   []string{"d", "e", "f"},
	}
	if got := info.Total(); got != 6 {
		t.Errorf("Total() = %d, want 6", got)
	}
}
