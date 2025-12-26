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

func TestHashProjectPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			// VERIFIED: echo -n "/Users/ashesh" | shasum -a 256
			path:     "/Users/ashesh",
			expected: "791e1ce1b3651ae5c05fc40e2ff27287a9a59008bcd7a449daf0cfb365d43bac",
		},
		{
			// VERIFIED: echo -n "/tmp/test" | shasum -a 256
			path:     "/tmp/test",
			expected: "0872effe487c8eb8681b0a627ce6f04c7a25bcd2a28834db42bdc40a52a85af1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := HashProjectPath(tt.path)
			if result != tt.expected {
				t.Errorf("HashProjectPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetGeminiSessionsDir(t *testing.T) {
	tmpDir := t.TempDir()
	geminiConfigDirOverride = tmpDir
	defer func() { geminiConfigDirOverride = "" }()

	projectPath := "/Users/ashesh/test-project"
	dir := GetGeminiSessionsDir(projectPath)

	hash := HashProjectPath(projectPath)
	expected := filepath.Join(tmpDir, "tmp", hash, "chats")

	if dir != expected {
		t.Errorf("GetGeminiSessionsDir(%q) = %q, want %q", projectPath, dir, expected)
	}
}
