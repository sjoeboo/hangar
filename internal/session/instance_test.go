package session

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestNewSessionStatusFlicker tests for green flicker on new session creation
// This reproduces the issue where a session briefly shows green before first poll
func TestNewSessionStatusFlicker(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Create a new session with a command (like user would do)
	inst := NewInstance("test-flicker", "/tmp")
	inst.Command = "echo hello" // Non-empty command

	// BEFORE Start() - should be idle
	if inst.Status != StatusIdle {
		t.Errorf("Before Start(): Status = %s, want idle", inst.Status)
	}

	// After Start() - current behavior sets StatusRunning immediately
	// This is the source of the flicker!
	err := inst.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = inst.Kill() }()

	t.Logf("After Start(): Status = %s", inst.Status)

	// Current behavior: StatusRunning is set in Start() if Command != ""
	// This causes a brief GREEN flash before the first GetStatus() poll
	if inst.Status == StatusRunning {
		t.Log("WARNING: FLICKER SOURCE - Status is 'running' immediately after Start()")
		t.Log("         This shows GREEN before the first tick updates it to the actual status")
	}

	// Simulate first tick (what happens 0-500ms after creation)
	err = inst.UpdateStatus()
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	t.Logf("After first UpdateStatus(): Status = %s", inst.Status)

	// After first poll, status should be 'waiting' (not 'running')
	// because GetStatus() returns "waiting" on first poll
	if inst.Status == StatusWaiting {
		t.Log("OK: First poll correctly shows 'waiting' (yellow)")
	}
}

// TestInstance_CanFork tests the CanFork method for Claude session forking
func TestInstance_CanFork(t *testing.T) {
	inst := NewInstance("test", "/tmp/test")

	// Without Claude session ID, cannot fork
	if inst.CanFork() {
		t.Error("CanFork() should be false without ClaudeSessionID")
	}

	// With Claude session ID, can fork
	inst.ClaudeSessionID = "abc-123-def"
	inst.ClaudeDetectedAt = time.Now()
	if !inst.CanFork() {
		t.Error("CanFork() should be true with recent ClaudeSessionID")
	}

	// With old detection time, cannot fork (stale)
	inst.ClaudeDetectedAt = time.Now().Add(-10 * time.Minute)
	if inst.CanFork() {
		t.Error("CanFork() should be false with stale ClaudeSessionID")
	}
}

// TestInstance_UpdateClaudeSession tests the UpdateClaudeSession method
func TestInstance_UpdateClaudeSession(t *testing.T) {
	inst := NewInstance("test", "/tmp/test")
	inst.Tool = "claude"

	// Mock: In real test, would need actual Claude running
	// For now, just test the method exists and doesn't crash
	inst.UpdateClaudeSession(nil)

	// After update with no Claude running, should have no session ID
	// (In integration test, would verify actual detection)
}

// TestInstance_Fork tests the Fork method
func TestInstance_Fork(t *testing.T) {
	inst := NewInstance("test", "/tmp/test")

	// Cannot fork without session ID
	_, err := inst.Fork("forked-test", "")
	if err == nil {
		t.Error("Fork() should fail without ClaudeSessionID")
	}

	// With session ID, Fork returns command to run
	inst.ClaudeSessionID = "abc-123"
	inst.ClaudeDetectedAt = time.Now()
	cmd, err := inst.Fork("forked-test", "")
	if err != nil {
		t.Errorf("Fork() failed: %v", err)
	}

	// Command should include CLAUDE_CONFIG_DIR and the session ID
	if !strings.Contains(cmd, "CLAUDE_CONFIG_DIR=") {
		t.Errorf("Fork() should set CLAUDE_CONFIG_DIR, got: %s", cmd)
	}
	if !strings.Contains(cmd, "--resume abc-123 --fork-session") {
		t.Errorf("Fork() should include resume and fork flags, got: %s", cmd)
	}
}

// TestInstance_CreateForkedInstance tests the CreateForkedInstance method
func TestInstance_CreateForkedInstance(t *testing.T) {
	inst := NewInstance("original", "/tmp/test")
	inst.GroupPath = "projects"

	// Cannot create fork without session ID
	_, _, err := inst.CreateForkedInstance("forked", "")
	if err == nil {
		t.Error("CreateForkedInstance() should fail without ClaudeSessionID")
	}

	// With session ID, creates new instance with fork command
	inst.ClaudeSessionID = "abc-123"
	inst.ClaudeDetectedAt = time.Now()
	forked, cmd, err := inst.CreateForkedInstance("forked", "")
	if err != nil {
		t.Errorf("CreateForkedInstance() failed: %v", err)
	}

	// Verify command includes config dir and fork flags
	if !strings.Contains(cmd, "CLAUDE_CONFIG_DIR=") {
		t.Errorf("Command should set CLAUDE_CONFIG_DIR, got: %s", cmd)
	}
	if !strings.Contains(cmd, "--resume abc-123 --fork-session") {
		t.Errorf("Command should include resume and fork flags, got: %s", cmd)
	}

	// Verify forked instance has correct properties
	if forked.Title != "forked" {
		t.Errorf("Forked title = %s, want forked", forked.Title)
	}
	if forked.ProjectPath != "/tmp/test" {
		t.Errorf("Forked path = %s, want /tmp/test", forked.ProjectPath)
	}
	if forked.GroupPath != "projects" {
		t.Errorf("Forked group = %s, want projects (inherited)", forked.GroupPath)
	}
	if !strings.Contains(forked.Command, "--resume abc-123 --fork-session") {
		t.Errorf("Forked command should include fork flags, got: %s", forked.Command)
	}
	if forked.Tool != "claude" {
		t.Errorf("Forked tool = %s, want claude", forked.Tool)
	}

	// Test with custom group path
	forked2, _, err := inst.CreateForkedInstance("forked2", "custom-group")
	if err != nil {
		t.Errorf("CreateForkedInstance() with custom group failed: %v", err)
	}
	if forked2.GroupPath != "custom-group" {
		t.Errorf("Forked group = %s, want custom-group", forked2.GroupPath)
	}
}

// TestNewInstanceWithTool tests that tools are set correctly without pre-assigned session IDs
func TestNewInstanceWithTool(t *testing.T) {
	// Shell tool should not have session ID (never will)
	shellInst := NewInstanceWithTool("shell-test", "/tmp/test", "shell")
	if shellInst.ClaudeSessionID != "" {
		t.Errorf("Shell session should not have ClaudeSessionID, got: %s", shellInst.ClaudeSessionID)
	}

	// Claude tool should NOT have pre-assigned ID (detection happens later)
	claudeInst := NewInstanceWithTool("claude-test", "/tmp/test", "claude")
	if claudeInst.ClaudeSessionID != "" {
		t.Errorf("Claude session should NOT have pre-assigned ClaudeSessionID (detection-based), got: %s", claudeInst.ClaudeSessionID)
	}
	if claudeInst.Tool != "claude" {
		t.Errorf("Tool = %s, want claude", claudeInst.Tool)
	}
	// ClaudeDetectedAt should be zero (detection hasn't happened yet)
	if !claudeInst.ClaudeDetectedAt.IsZero() {
		t.Error("ClaudeDetectedAt should be zero until detection happens")
	}
}

// TestBuildClaudeCommand tests that claude command is built with capture-resume pattern
func TestBuildClaudeCommand(t *testing.T) {
	inst := NewInstanceWithTool("test", "/tmp/test", "claude")

	// Test with simple "claude" command
	cmd := inst.buildClaudeCommand("claude")

	// Should contain CLAUDE_CONFIG_DIR (appears twice: once for capture, once for resume)
	if !strings.Contains(cmd, "CLAUDE_CONFIG_DIR=") {
		t.Errorf("Should contain CLAUDE_CONFIG_DIR, got: %s", cmd)
	}

	// Should use capture-resume pattern: -p "." --output-format json
	if !strings.Contains(cmd, `-p "."`) {
		t.Errorf("Should contain -p \".\" for print mode, got: %s", cmd)
	}
	if !strings.Contains(cmd, "--output-format json") {
		t.Errorf("Should contain --output-format json, got: %s", cmd)
	}

	// Should store session ID in tmux environment
	if !strings.Contains(cmd, "tmux set-environment CLAUDE_SESSION_ID") {
		t.Errorf("Should store session ID in tmux env, got: %s", cmd)
	}

	// Should resume the captured session
	if !strings.Contains(cmd, `--resume "$session_id"`) {
		t.Errorf("Should resume the captured session ID, got: %s", cmd)
	}

	// Should contain dangerously-skip-permissions
	if !strings.Contains(cmd, "--dangerously-skip-permissions") {
		t.Errorf("Should contain --dangerously-skip-permissions, got: %s", cmd)
	}

	// Test with non-claude tool (should not modify)
	shellInst := NewInstance("shell-test", "/tmp/test")
	shellCmd := shellInst.buildClaudeCommand("bash")
	if shellCmd != "bash" {
		t.Errorf("Non-claude command should not be modified, got: %s", shellCmd)
	}
}

// TestCreateForkedInstance_NoSessionIDFlag tests that forked sessions
// do NOT use --session-id flag (incompatible with --resume)
func TestCreateForkedInstance_NoSessionIDFlag(t *testing.T) {
	inst := NewInstance("original", "/tmp/test")
	inst.ClaudeSessionID = "parent-abc-123"
	inst.ClaudeDetectedAt = time.Now()

	forked, cmd, err := inst.CreateForkedInstance("forked", "")
	if err != nil {
		t.Fatalf("CreateForkedInstance() failed: %v", err)
	}

	// Command should NOT contain --session-id (incompatible with --resume)
	if strings.Contains(cmd, "--session-id") {
		t.Errorf("Fork command should NOT contain --session-id (incompatible with --resume), got: %s", cmd)
	}

	// Command SHOULD contain --resume and --fork-session
	if !strings.Contains(cmd, "--resume parent-abc-123") {
		t.Errorf("Fork command should contain --resume with parent ID, got: %s", cmd)
	}
	if !strings.Contains(cmd, "--fork-session") {
		t.Errorf("Fork command should contain --fork-session, got: %s", cmd)
	}

	// Forked instance should have empty ClaudeSessionID (will be detected later)
	if forked.ClaudeSessionID != "" {
		t.Errorf("Forked instance should have empty ClaudeSessionID (detected later), got: %s", forked.ClaudeSessionID)
	}

	// Forked instance should have fork flag in command
	if !strings.Contains(forked.Command, "--fork-session") {
		t.Errorf("Forked instance Command should contain --fork-session, got: %s", forked.Command)
	}
}

// TestWaitForClaudeSession tests the wait-for-detection functionality
func TestWaitForClaudeSession(t *testing.T) {
	inst := NewInstance("test", "/tmp/nonexistent-project-dir")
	inst.Tool = "claude"

	// Should timeout and return empty when no session file exists
	start := time.Now()
	sessionID := inst.WaitForClaudeSession(500 * time.Millisecond)
	elapsed := time.Since(start)

	if sessionID != "" {
		t.Errorf("Should return empty when no session file, got: %s", sessionID)
	}

	// Should have waited at least close to the timeout
	if elapsed < 400*time.Millisecond {
		t.Errorf("Should have waited ~500ms, but only waited %v", elapsed)
	}

	// ClaudeSessionID should still be empty
	if inst.ClaudeSessionID != "" {
		t.Errorf("ClaudeSessionID should be empty, got: %s", inst.ClaudeSessionID)
	}
}

func TestInstance_GetSessionIDFromTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Create instance with tmux session
	inst := NewInstanceWithTool("tmux-env-test", "/tmp", "claude")

	// Start the session
	err := inst.Start()
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}
	defer inst.Kill()

	// Initially should return empty (no CLAUDE_SESSION_ID set)
	if id := inst.GetSessionIDFromTmux(); id != "" {
		t.Errorf("GetSessionIDFromTmux should return empty initially, got: %s", id)
	}

	// Set the environment variable directly via tmux
	tmuxSess := inst.GetTmuxSession()
	if tmuxSess == nil {
		t.Fatal("tmux session is nil")
	}

	testSessionID := "test-uuid-12345"
	err = tmuxSess.SetEnvironment("CLAUDE_SESSION_ID", testSessionID)
	if err != nil {
		t.Fatalf("Failed to set environment: %v", err)
	}

	// Now should return the session ID
	if id := inst.GetSessionIDFromTmux(); id != testSessionID {
		t.Errorf("GetSessionIDFromTmux = %q, want %q", id, testSessionID)
	}
}

func TestInstance_UpdateClaudeSession_TmuxFirst(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Create and start instance
	inst := NewInstanceWithTool("update-test", "/tmp", "claude")
	err := inst.Start()
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}
	defer inst.Kill()

	// Set session ID in tmux environment
	testSessionID := "tmux-session-abc123"
	tmuxSess := inst.GetTmuxSession()
	err = tmuxSess.SetEnvironment("CLAUDE_SESSION_ID", testSessionID)
	if err != nil {
		t.Fatalf("Failed to set environment: %v", err)
	}

	// Clear any existing detection
	inst.ClaudeSessionID = ""
	inst.ClaudeDetectedAt = time.Time{}

	// Call UpdateClaudeSession
	inst.UpdateClaudeSession(nil)

	// Should have picked up from tmux environment
	if inst.ClaudeSessionID != testSessionID {
		t.Errorf("ClaudeSessionID = %q, want %q (from tmux env)", inst.ClaudeSessionID, testSessionID)
	}
}
