package session

import (
	"os"
	"path/filepath"
	"testing"
)

// setupProjectsTest redirects HOME to a temp dir so LoadProjects/SaveProjects
// operate on a throwaway ~/.hangar/projects.toml.
func setupProjectsTest(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", originalHome) })

	hangarDir := filepath.Join(tempDir, ".hangar")
	if err := os.MkdirAll(hangarDir, 0755); err != nil {
		t.Fatalf("failed to create hangar dir: %v", err)
	}
	return tempDir
}

// ============================================================================
// RenameProject tests
// ============================================================================

func TestRenameProject_Success(t *testing.T) {
	setupProjectsTest(t)

	// Seed a project
	if err := AddProject("myproject", "/tmp/repo", "main"); err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	// Rename it
	if err := RenameProject("myproject", "newname"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	// Old name must not exist
	_, err := GetProject("myproject")
	if err == nil {
		t.Error("GetProject(oldName) should return error after rename")
	}

	// New name must exist and preserve other fields
	p, err := GetProject("newname")
	if err != nil {
		t.Fatalf("GetProject(newName): %v", err)
	}
	if p.Name != "newname" {
		t.Errorf("Name: got %q, want %q", p.Name, "newname")
	}
	if p.BaseDir != "/tmp/repo" {
		t.Errorf("BaseDir: got %q, want %q", p.BaseDir, "/tmp/repo")
	}
	if p.BaseBranch != "main" {
		t.Errorf("BaseBranch: got %q, want %q", p.BaseBranch, "main")
	}
}

func TestRenameProject_CaseInsensitiveMatch(t *testing.T) {
	setupProjectsTest(t)

	if err := AddProject("CaseTest", "/tmp/repo", "master"); err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	// Match by different case
	if err := RenameProject("CASETEST", "renamed"); err != nil {
		t.Fatalf("RenameProject with different case: %v", err)
	}

	p, err := GetProject("renamed")
	if err != nil {
		t.Fatalf("GetProject(renamed): %v", err)
	}
	if p.Name != "renamed" {
		t.Errorf("Name: got %q, want %q", p.Name, "renamed")
	}
}

func TestRenameProject_NotFound(t *testing.T) {
	setupProjectsTest(t)

	err := RenameProject("doesnotexist", "newname")
	if err == nil {
		t.Fatal("RenameProject should return error when project not found")
	}
}

func TestRenameProject_PreservesOtherProjects(t *testing.T) {
	setupProjectsTest(t)

	if err := AddProject("alpha", "/tmp/alpha", "main"); err != nil {
		t.Fatalf("AddProject alpha: %v", err)
	}
	if err := AddProject("beta", "/tmp/beta", "master"); err != nil {
		t.Fatalf("AddProject beta: %v", err)
	}

	if err := RenameProject("alpha", "gamma"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	// beta must still exist
	p, err := GetProject("beta")
	if err != nil {
		t.Fatalf("GetProject(beta): %v", err)
	}
	if p.BaseDir != "/tmp/beta" {
		t.Errorf("beta BaseDir: got %q, want %q", p.BaseDir, "/tmp/beta")
	}

	// gamma must exist
	if _, err := GetProject("gamma"); err != nil {
		t.Fatalf("GetProject(gamma): %v", err)
	}

	// alpha must be gone
	if _, err := GetProject("alpha"); err == nil {
		t.Error("GetProject(alpha) should return error after rename")
	}
}

func TestRenameProject_PreservesOrder(t *testing.T) {
	setupProjectsTest(t)

	if err := AddProject("first", "/tmp/first", "main"); err != nil {
		t.Fatalf("AddProject first: %v", err)
	}
	if err := AddProject("second", "/tmp/second", "main"); err != nil {
		t.Fatalf("AddProject second: %v", err)
	}

	// Capture original order for "second"
	orig, err := GetProject("second")
	if err != nil {
		t.Fatalf("GetProject second: %v", err)
	}
	origOrder := orig.Order

	if err := RenameProject("second", "renamed-second"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	p, err := GetProject("renamed-second")
	if err != nil {
		t.Fatalf("GetProject renamed-second: %v", err)
	}
	if p.Order != origOrder {
		t.Errorf("Order: got %d, want %d (should be preserved)", p.Order, origOrder)
	}
}

func TestRenameProject_EmptyNewName(t *testing.T) {
	setupProjectsTest(t)

	err := RenameProject("anything", "")
	if err == nil {
		t.Error("expected error for empty new name")
	}
}

func TestRenameProject_ConflictWithExisting(t *testing.T) {
	setupProjectsTest(t)

	// Seed two projects
	if err := AddProject("Alpha", "/tmp/alpha", "main"); err != nil {
		t.Fatalf("AddProject Alpha: %v", err)
	}
	if err := AddProject("Beta", "/tmp/beta", "main"); err != nil {
		t.Fatalf("AddProject Beta: %v", err)
	}

	// Attempt to rename "Alpha" to "beta" — case-insensitive collision with "Beta"
	err := RenameProject("Alpha", "beta")
	if err == nil {
		t.Fatal("RenameProject should return error when new name collides with existing project")
	}

	// Verify "Alpha" was NOT renamed
	p, err := GetProject("Alpha")
	if err != nil {
		t.Fatalf("GetProject(Alpha) should still exist after failed rename: %v", err)
	}
	if p.Name != "Alpha" {
		t.Errorf("Alpha name: got %q, want %q", p.Name, "Alpha")
	}
}
