package session

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Force test profile to prevent production data corruption
	// See CLAUDE.md: "2025-12-11 Incident: Tests with AGENTDECK_PROFILE=work overwrote ALL 36 production sessions"
	os.Setenv("AGENTDECK_PROFILE", "_test")

	// Run tests
	code := m.Run()

	// Cleanup: Kill any orphaned test sessions after tests complete
	// This prevents RAM waste from lingering test sessions
	// See CLAUDE.md: "2026-01-20 Incident: 20+ Test-Skip-Regen sessions orphaned, wasting ~3GB RAM"
	cleanupTestSessions()

	os.Exit(code)
}

// cleanupTestSessions kills any tmux sessions created during testing
func cleanupTestSessions() {
	// Get all tmux sessions
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return // No tmux or no sessions
	}

	sessions := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, sess := range sessions {
		// Kill test sessions (Test-Skip-Regen, test_, _test patterns)
		if strings.Contains(sess, "Test-Skip-Regen") ||
			strings.Contains(sess, "test_") ||
			strings.HasPrefix(sess, "agentdeck_test") {
			_ = exec.Command("tmux", "kill-session", "-t", sess).Run()
		}
	}
}
