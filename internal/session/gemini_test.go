package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetGeminiConfigDir_Default(t *testing.T) {
	// Clear override
	geminiConfigDirOverride = ""

	dir := GetGeminiConfigDir()

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".gemini")

	if dir != expected {
		t.Errorf("GetGeminiConfigDir() = %q, want %q", dir, expected)
	}
}

func TestGetGeminiConfigDir_Override(t *testing.T) {
	// Set override for testing
	tmpDir := t.TempDir()
	geminiConfigDirOverride = tmpDir
	defer func() { geminiConfigDirOverride = "" }()

	dir := GetGeminiConfigDir()

	if dir != tmpDir {
		t.Errorf("GetGeminiConfigDir() = %q, want %q", dir, tmpDir)
	}
}
