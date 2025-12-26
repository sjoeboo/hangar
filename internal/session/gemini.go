package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
	if projectHash == "" {
		return "" // Cannot determine sessions dir without valid hash
	}
	return filepath.Join(configDir, "tmp", projectHash, "chats")
}

// GeminiSessionInfo holds parsed session metadata
type GeminiSessionInfo struct {
	SessionID    string    // Full UUID
	Filename     string    // session-2025-12-26T15-09-4d8fcb4d.json
	StartTime    time.Time
	LastUpdated  time.Time
	MessageCount int
}

// parseGeminiSessionFile reads a session file and extracts metadata
// VERIFIED: Field names use camelCase (sessionId, not session_id)
func parseGeminiSessionFile(filePath string) (GeminiSessionInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return GeminiSessionInfo{}, err
	}

	var session struct {
		SessionID   string            `json:"sessionId"` // VERIFIED: camelCase
		StartTime   string            `json:"startTime"`
		LastUpdated string            `json:"lastUpdated"`
		Messages    []json.RawMessage `json:"messages"`
	}

	if err := json.Unmarshal(data, &session); err != nil {
		return GeminiSessionInfo{}, fmt.Errorf("failed to parse session: %w", err)
	}

	startTime, _ := time.Parse(time.RFC3339, session.StartTime)
	lastUpdated, _ := time.Parse(time.RFC3339, session.LastUpdated)

	return GeminiSessionInfo{
		SessionID:    session.SessionID,
		Filename:     filepath.Base(filePath),
		StartTime:    startTime,
		LastUpdated:  lastUpdated,
		MessageCount: len(session.Messages),
	}, nil
}
