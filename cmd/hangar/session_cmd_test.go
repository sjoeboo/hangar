package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjoeboo/hangar/internal/session"
)

func TestMCPInfoForJSON_NilOrEmpty(t *testing.T) {
	if got := mcpInfoForJSON(nil); got != nil {
		t.Fatalf("mcpInfoForJSON(nil) = %#v, want nil", got)
	}

	if got := mcpInfoForJSON(&session.MCPInfo{}); got != nil {
		t.Fatalf("mcpInfoForJSON(empty) = %#v, want nil", got)
	}
}

func TestMCPInfoForJSON_UsesSlicesAndIsMarshalable(t *testing.T) {
	info := &session.MCPInfo{
		Global:  []string{"global-a"},
		Project: []string{"project-a"},
		LocalMCPs: []session.LocalMCP{
			{Name: "local-a", SourcePath: "/tmp"},
		},
	}

	got := mcpInfoForJSON(info)
	if got == nil {
		t.Fatal("mcpInfoForJSON returned nil for populated MCP info")
	}

	local, ok := got["local"].([]string)
	if !ok {
		t.Fatalf("mcps.local type = %T, want []string", got["local"])
	}
	if len(local) != 1 || local[0] != "local-a" {
		t.Fatalf("mcps.local = %#v, want []string{\"local-a\"}", local)
	}

	payload := map[string]interface{}{"mcps": got}
	if _, err := json.Marshal(payload); err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
}

// =============================================================================
// Tests for session set path validation logic
// =============================================================================

func TestValidateDirectoryPath(t *testing.T) {
	t.Run("empty string is rejected", func(t *testing.T) {
		err := validateDirectoryPath("")
		if err == nil {
			t.Error("expected error for empty path, got nil")
		}
	})

	t.Run("nonexistent path is rejected", func(t *testing.T) {
		err := validateDirectoryPath("/nonexistent/path/xyz/abc")
		if err == nil {
			t.Error("expected error for nonexistent path, got nil")
		}
	})

	t.Run("existing directory is accepted", func(t *testing.T) {
		dir := t.TempDir()
		err := validateDirectoryPath(dir)
		if err != nil {
			t.Errorf("expected nil for valid directory, got: %v", err)
		}
	})

	t.Run("file path is rejected", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "somefile.txt")
		if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		err := validateDirectoryPath(filePath)
		if err == nil {
			t.Error("expected error for file path, got nil")
		}
	})
}

// TestSessionSetPath_RejectsNonexistentPath is an integration test that
// exercises the full runSessionSet command path end-to-end.
//
// It validates that `hangar session set <id> path <value>` rejects
// non-existent paths with a non-zero exit code. This test requires a running
// hangar storage instance; run manually:
//
//	hangar session set <id> path /nonexistent/path/xyz
func TestSessionSetPath_RejectsNonexistentPath(t *testing.T) {
	t.Skip("integration test: run manually with: hangar session set <id> path /nonexistent")
}
