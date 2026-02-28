package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectClaudeHooks_Fresh(t *testing.T) {
	tmpDir := t.TempDir()

	installed, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("InjectClaudeHooks failed: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be newly installed")
	}

	// Read settings.json and verify hooks are present
	data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		t.Fatal("settings.json missing 'hooks' key")
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		t.Fatalf("Failed to parse hooks: %v", err)
	}

	// Verify all expected events are present
	expectedEvents := []string{"SessionStart", "UserPromptSubmit", "Stop", "PermissionRequest", "Notification", "SessionEnd"}
	for _, event := range expectedEvents {
		if _, ok := hooks[event]; !ok {
			t.Errorf("Missing hook event: %s", event)
		}
	}

	// Verify the hook command is correct
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(hooks["SessionStart"], &matchers); err != nil {
		t.Fatalf("Failed to parse SessionStart matchers: %v", err)
	}
	if len(matchers) == 0 {
		t.Fatal("SessionStart has no matchers")
	}
	if len(matchers[0].Hooks) == 0 {
		t.Fatal("SessionStart matcher has no hooks")
	}
	if matchers[0].Hooks[0].Command != hangarHookCommand {
		t.Errorf("Hook command = %q, want %q", matchers[0].Hooks[0].Command, hangarHookCommand)
	}
	if !matchers[0].Hooks[0].Async {
		t.Error("Hook should be async")
	}
}

func TestInjectClaudeHooks_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Write existing settings with a custom setting and user hook
	existing := map[string]json.RawMessage{
		"apiKey": json.RawMessage(`"sk-test-123"`),
		"hooks": json.RawMessage(`{
			"SessionStart": [{"hooks": [{"type": "command", "command": "my-custom-hook"}]}]
		}`),
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	installed, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("InjectClaudeHooks failed: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be installed")
	}

	// Verify existing setting is preserved
	readData, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(readData, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	if string(settings["apiKey"]) != `"sk-test-123"` {
		t.Errorf("apiKey was not preserved: %s", settings["apiKey"])
	}

	// Verify user hook is preserved alongside hangar hook
	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("Failed to parse hooks: %v", err)
	}

	var matchers []claudeHookMatcher
	if err := json.Unmarshal(hooks["SessionStart"], &matchers); err != nil {
		t.Fatalf("Failed to parse SessionStart matchers: %v", err)
	}

	// Should have the original matcher with user hook, plus hangar's hook appended
	foundCustom := false
	foundHangar := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Command == "my-custom-hook" {
				foundCustom = true
			}
			if h.Command == hangarHookCommand {
				foundHangar = true
			}
		}
	}

	if !foundCustom {
		t.Error("User's custom hook was not preserved")
	}
	if !foundHangar {
		t.Error("Hangar hook was not added")
	}
}

func TestInjectClaudeHooks_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// First install
	installed1, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("First install failed: %v", err)
	}
	if !installed1 {
		t.Error("First install should return true")
	}

	// Second install should be a no-op
	installed2, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}
	if installed2 {
		t.Error("Second install should return false (already installed)")
	}

	// Verify no duplicate hooks
	data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("Failed to parse hooks: %v", err)
	}

	var matchers []claudeHookMatcher
	if err := json.Unmarshal(hooks["SessionStart"], &matchers); err != nil {
		t.Fatalf("Failed to parse SessionStart matchers: %v", err)
	}

	hookCount := 0
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Command == hangarHookCommand {
				hookCount++
			}
		}
	}
	if hookCount != 1 {
		t.Errorf("Expected 1 hangar hook, got %d (duplication bug)", hookCount)
	}
}

func TestRemoveClaudeHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// Install first
	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("InjectClaudeHooks failed: %v", err)
	}

	// Remove
	removed, err := RemoveClaudeHooks(tmpDir)
	if err != nil {
		t.Fatalf("RemoveClaudeHooks failed: %v", err)
	}
	if !removed {
		t.Error("Expected hooks to be removed")
	}

	// Verify hooks are gone
	if CheckClaudeHooksInstalled(tmpDir) {
		t.Error("Hooks should not be installed after removal")
	}
}

func TestRemoveClaudeHooks_PreservesUserHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// Write settings with both user and hangar hooks
	existing := map[string]json.RawMessage{
		"hooks": json.RawMessage(`{
			"SessionStart": [
				{"hooks": [{"type": "command", "command": "my-custom-hook"}, {"type": "command", "command": "hangar hook-handler", "async": true}]}
			]
		}`),
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	// Remove hangar hooks
	removed, err := RemoveClaudeHooks(tmpDir)
	if err != nil {
		t.Fatalf("RemoveClaudeHooks failed: %v", err)
	}
	if !removed {
		t.Error("Expected hooks to be removed")
	}

	// Verify user hook is preserved
	readData, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(readData, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("Failed to parse hooks: %v", err)
	}

	var matchers []claudeHookMatcher
	if err := json.Unmarshal(hooks["SessionStart"], &matchers); err != nil {
		t.Fatalf("Failed to parse SessionStart matchers: %v", err)
	}

	foundCustom := false
	foundHangar := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Command == "my-custom-hook" {
				foundCustom = true
			}
			if h.Command == hangarHookCommand {
				foundHangar = true
			}
		}
	}

	if !foundCustom {
		t.Error("User hook should be preserved")
	}
	if foundHangar {
		t.Error("Hangar hook should be removed")
	}
}

func TestCheckClaudeHooksInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	// Not installed yet
	if CheckClaudeHooksInstalled(tmpDir) {
		t.Error("Hooks should not be installed initially")
	}

	// Install
	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("InjectClaudeHooks failed: %v", err)
	}

	// Should be installed
	if !CheckClaudeHooksInstalled(tmpDir) {
		t.Error("Hooks should be installed after InjectClaudeHooks")
	}

	// Remove
	if _, err := RemoveClaudeHooks(tmpDir); err != nil {
		t.Fatalf("RemoveClaudeHooks failed: %v", err)
	}

	// Should not be installed
	if CheckClaudeHooksInstalled(tmpDir) {
		t.Error("Hooks should not be installed after RemoveClaudeHooks")
	}
}

func TestParseClaudeVersion(t *testing.T) {
	tests := []struct {
		output  string
		wantVer string
		wantErr bool
	}{
		{"claude 2.1.63 (Claude Code)", "2.1.63", false},
		{"claude 1.0.0 (Claude Code)", "1.0.0", false},
		{"2.1.63", "2.1.63", false},
		{"Claude Code v2.1.63", "2.1.63", false},
		{"claude 2.1.63 (Claude Code)\nUpdate available: 2.2.0", "2.1.63", false},
		{"no version here", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got, err := parseClaudeVersion(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseClaudeVersion(%q): expected error, got %q", tt.output, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseClaudeVersion(%q): unexpected error: %v", tt.output, err)
			}
			if got != tt.wantVer {
				t.Errorf("parseClaudeVersion(%q) = %q, want %q", tt.output, got, tt.wantVer)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		major   int
		minor   int
		patch   int
		want    bool
	}{
		{"2.1.63", 2, 1, 63, true},
		{"2.1.64", 2, 1, 63, true},
		{"2.2.0", 2, 1, 63, true},
		{"3.0.0", 2, 1, 63, true},
		{"2.1.62", 2, 1, 63, false},
		{"2.0.99", 2, 1, 63, false},
		{"1.9.99", 2, 1, 63, false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := versionAtLeast(tt.version, tt.major, tt.minor, tt.patch)
			if got != tt.want {
				t.Errorf("versionAtLeast(%q, %d, %d, %d) = %v, want %v",
					tt.version, tt.major, tt.minor, tt.patch, got, tt.want)
			}
		})
	}
}

func TestClaudeSupportsHTTPHooks(t *testing.T) {
	if claudeSupportsHTTPHooks("2.1.63") != true {
		t.Error("2.1.63 should support HTTP hooks")
	}
	if claudeSupportsHTTPHooks("2.1.62") != false {
		t.Error("2.1.62 should not support HTTP hooks")
	}
	if claudeSupportsHTTPHooks("") != false {
		t.Error("empty version should not support HTTP hooks")
	}
}

func TestInjectClaudeHooks_CommandType_WhenPortZero(t *testing.T) {
	tmpDir := t.TempDir()

	installed, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("InjectClaudeHooks: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be installed")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Type == "http" {
				t.Error("Should not inject HTTP hook when port=0")
			}
		}
	}
}

func TestInjectClaudeHooks_HTTPType_WhenPortNonZero(t *testing.T) {
	tmpDir := t.TempDir()

	installed, err := InjectClaudeHooks(tmpDir, 2437)
	if err != nil {
		t.Fatalf("InjectClaudeHooks: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be installed")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	foundHTTP := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Type == "http" {
				foundHTTP = true
				wantURL := "http://127.0.0.1:2437/hooks"
				if h.URL != wantURL {
					t.Errorf("HTTP hook URL = %q, want %q", h.URL, wantURL)
				}
				if h.Headers["X-Hangar-Instance-Id"] != "$HANGAR_INSTANCE_ID" {
					t.Errorf("Headers[X-Hangar-Instance-Id] = %q, want %q",
						h.Headers["X-Hangar-Instance-Id"], "$HANGAR_INSTANCE_ID")
				}
				if len(h.AllowedEnvVars) != 1 || h.AllowedEnvVars[0] != "HANGAR_INSTANCE_ID" {
					t.Errorf("AllowedEnvVars = %v, want [HANGAR_INSTANCE_ID]", h.AllowedEnvVars)
				}
				if h.Timeout != 5 {
					t.Errorf("Timeout = %d, want 5", h.Timeout)
				}
			}
		}
	}
	if !foundHTTP {
		t.Error("Expected HTTP hook entry when port=2437")
	}
}

func TestInjectClaudeHooks_UpgradeCommandToHTTP(t *testing.T) {
	tmpDir := t.TempDir()

	// First: install command hooks
	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("command install: %v", err)
	}

	// Now: upgrade to HTTP hooks
	upgraded, err := InjectClaudeHooks(tmpDir, 2437)
	if err != nil {
		t.Fatalf("http upgrade: %v", err)
	}
	if !upgraded {
		t.Error("Expected upgrade to return installed=true")
	}

	// Verify command hooks removed, HTTP hooks present
	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	foundCommand := false
	foundHTTP := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Command == hangarHookCommand {
				foundCommand = true
			}
			if h.Type == "http" {
				foundHTTP = true
			}
		}
	}
	if foundCommand {
		t.Error("Command hook should be removed after HTTP upgrade")
	}
	if !foundHTTP {
		t.Error("HTTP hook should be present after upgrade")
	}
}

func TestCheckClaudeHTTPHooksInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	// Not installed initially
	if CheckClaudeHTTPHooksInstalled(tmpDir) {
		t.Error("Should not be installed initially")
	}

	// Install command hooks — should NOT count as HTTP
	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("command install: %v", err)
	}
	if CheckClaudeHTTPHooksInstalled(tmpDir) {
		t.Error("Command hooks should not be reported as HTTP hooks")
	}

	// Upgrade to HTTP hooks
	if _, err := InjectClaudeHooks(tmpDir, 2437); err != nil {
		t.Fatalf("http upgrade: %v", err)
	}
	if !CheckClaudeHTTPHooksInstalled(tmpDir) {
		t.Error("HTTP hooks should be reported as HTTP after upgrade")
	}
}

func TestNotificationMatcher(t *testing.T) {
	tmpDir := t.TempDir()

	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("InjectClaudeHooks failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("Failed to parse hooks: %v", err)
	}

	// Notification event should have a matcher pattern
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(hooks["Notification"], &matchers); err != nil {
		t.Fatalf("Failed to parse Notification matchers: %v", err)
	}

	if len(matchers) == 0 {
		t.Fatal("Notification has no matchers")
	}
	if matchers[0].Matcher != "permission_prompt|elicitation_dialog" {
		t.Errorf("Notification matcher = %q, want %q", matchers[0].Matcher, "permission_prompt|elicitation_dialog")
	}
}
