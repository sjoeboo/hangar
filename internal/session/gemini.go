package session

import (
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
