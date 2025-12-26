package session

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// geminiConfigDirOverride allows tests to override config directory
var geminiConfigDirOverride string

// GetGeminiConfigDir returns ~/.gemini
// Unlike Claude, Gemini has no GEMINI_CONFIG_DIR env var override
func GetGeminiConfigDir() string {
	if geminiConfigDirOverride != "" {
		return geminiConfigDirOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini")
}

// HashProjectPath generates SHA256 hash of absolute project path
// This matches Gemini CLI's project hash algorithm for session storage
// VERIFIED: echo -n "/Users/ashesh" | shasum -a 256
func HashProjectPath(projectPath string) string {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:])
}

// GetGeminiSessionsDir returns the chats directory for a project
// Format: ~/.gemini/tmp/<project_hash>/chats/
func GetGeminiSessionsDir(projectPath string) string {
	configDir := GetGeminiConfigDir()
	projectHash := HashProjectPath(projectPath)
	return filepath.Join(configDir, "tmp", projectHash, "chats")
}
