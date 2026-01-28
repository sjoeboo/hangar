package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStorageUpdatedAtTimestamp verifies that SaveWithGroups sets the UpdatedAt timestamp
// and GetUpdatedAt() returns it correctly.
func TestStorageUpdatedAtTimestamp(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "sessions.json")

	// Create storage instance
	s := &Storage{
		path:    storagePath,
		profile: "_test",
	}

	// Create test data
	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test Session",
			ProjectPath: "/tmp/test",
			GroupPath:   "test-group",
			Command:     "claude",
			Tool:        "claude",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	// Save data
	beforeSave := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp differs

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp differs
	afterSave := time.Now()

	// Get the updated timestamp
	updatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Fatalf("GetUpdatedAt failed: %v", err)
	}

	// Verify timestamp is within expected range
	if updatedAt.Before(beforeSave) {
		t.Errorf("UpdatedAt %v is before save started %v", updatedAt, beforeSave)
	}
	if updatedAt.After(afterSave) {
		t.Errorf("UpdatedAt %v is after save completed %v", updatedAt, afterSave)
	}

	// Verify timestamp is not zero
	if updatedAt.IsZero() {
		t.Error("UpdatedAt is zero, expected a valid timestamp")
	}

	// Save again and verify timestamp updates
	time.Sleep(50 * time.Millisecond)
	firstUpdatedAt := updatedAt

	err = s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("Second SaveWithGroups failed: %v", err)
	}

	secondUpdatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Fatalf("Second GetUpdatedAt failed: %v", err)
	}

	// Verify second timestamp is after first
	if !secondUpdatedAt.After(firstUpdatedAt) {
		t.Errorf("Second UpdatedAt %v should be after first %v", secondUpdatedAt, firstUpdatedAt)
	}
}

// TestGetUpdatedAtNoFile verifies behavior when storage file doesn't exist
func TestGetUpdatedAtNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "nonexistent.json")

	s := &Storage{
		path:    storagePath,
		profile: "_test",
	}

	_, err := s.GetUpdatedAt()
	if err == nil {
		t.Error("Expected error when file doesn't exist, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected IsNotExist error, got: %v", err)
	}
}

// TestLoadLite verifies that LoadLite returns raw InstanceData without tmux initialization
func TestLoadLite(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "sessions.json")

	// Create storage instance
	s := &Storage{
		path:    storagePath,
		profile: "_test",
	}

	// Create test data with tmux session name
	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test Session 1",
			ProjectPath: "/tmp/test1",
			GroupPath:   "test-group",
			Command:     "claude",
			Tool:        "claude",
			Status:      StatusWaiting,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-2",
			Title:       "Test Session 2",
			ProjectPath: "/tmp/test2",
			GroupPath:   "other-group",
			Command:     "gemini",
			Tool:        "gemini",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	// Set tmux session names manually (simulating what Storage.convertToInstances would set)
	// In production, this would be set by tmux.NewSession
	instances[0].tmuxSession = nil // Simulate no tmux connection
	instances[1].tmuxSession = nil

	// Save data
	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// LoadLite should return raw InstanceData without tmux initialization
	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Fatalf("LoadLite failed: %v", err)
	}

	// Verify we got the right number of instances
	if len(instData) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instData))
	}

	// Verify instance data is correct
	if instData[0].ID != "test-1" {
		t.Errorf("Expected first instance ID 'test-1', got '%s'", instData[0].ID)
	}
	if instData[0].Title != "Test Session 1" {
		t.Errorf("Expected first instance title 'Test Session 1', got '%s'", instData[0].Title)
	}
	if instData[0].Status != StatusWaiting {
		t.Errorf("Expected first instance status 'waiting', got '%s'", instData[0].Status)
	}

	if instData[1].ID != "test-2" {
		t.Errorf("Expected second instance ID 'test-2', got '%s'", instData[1].ID)
	}
	if instData[1].Tool != "gemini" {
		t.Errorf("Expected second instance tool 'gemini', got '%s'", instData[1].Tool)
	}

	// Groups should be empty or nil since we didn't create any
	if len(groupData) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groupData))
	}
}

// TestLoadLiteNoFile verifies LoadLite returns empty slice when file doesn't exist
func TestLoadLiteNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "nonexistent.json")

	s := &Storage{
		path:    storagePath,
		profile: "_test",
	}

	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Errorf("LoadLite should not return error for non-existent file, got: %v", err)
	}
	if len(instData) != 0 {
		t.Errorf("Expected empty instances, got %d", len(instData))
	}
	if len(groupData) != 0 {
		t.Errorf("Expected empty groups, got %d", len(groupData))
	}
}
