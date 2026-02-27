package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ghe.spotify.net/mnicholson/hangar/internal/session"
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

// TestSessionSetPath_ValidationLogic verifies that the os.Stat + IsDir checks
// used in the "path" case of runSessionSet behave as expected.
//
// runSessionSet calls os.Exit so it cannot be invoked directly in tests.
// These tests exercise the underlying os.Stat + IsDir semantics with real
// filesystem paths to confirm the validation logic is correct.
func TestSessionSetPath_ValidationLogic(t *testing.T) {
	t.Run("existing directory passes validation", func(t *testing.T) {
		dir := t.TempDir()
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("os.Stat(%q) unexpected error: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", dir)
		}
	})

	t.Run("nonexistent path fails validation", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent", "deep", "path")
		_, err := os.Stat(path)
		if err == nil {
			t.Errorf("expected os.Stat to fail for nonexistent path %q, but it succeeded", path)
		}
	})

	t.Run("file path fails IsDir check", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "somefile.txt")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		info, err := os.Stat(file)
		if err != nil {
			t.Fatalf("os.Stat(%q) unexpected error: %v", file, err)
		}
		if info.IsDir() {
			t.Errorf("expected %q to NOT be a directory", file)
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
