package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/statedb"
)

// newTestStorage creates a Storage backed by an in-memory-like temp dir SQLite database.
func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")
	db, err := statedb.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &Storage{db: db, dbPath: dbPath, profile: "_test"}
}

// TestStorageUpdatedAtTimestamp verifies that SaveWithGroups sets the UpdatedAt timestamp
// and GetUpdatedAt() returns it correctly.
func TestStorageUpdatedAtTimestamp(t *testing.T) {
	s := newTestStorage(t)

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
	time.Sleep(10 * time.Millisecond)

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
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

// TestGetUpdatedAtEmpty verifies behavior when no data has been saved
func TestGetUpdatedAtEmpty(t *testing.T) {
	s := newTestStorage(t)

	updatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !updatedAt.IsZero() {
		t.Errorf("Expected zero time for empty db, got %v", updatedAt)
	}
}

// TestLoadLite verifies that LoadLite returns raw InstanceData without tmux initialization
func TestLoadLite(t *testing.T) {
	s := newTestStorage(t)

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

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Fatalf("LoadLite failed: %v", err)
	}

	if len(instData) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instData))
	}

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

	if len(groupData) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groupData))
	}
}

// TestLoadLiteEmptyDB verifies LoadLite returns empty slice when database is empty
func TestLoadLiteEmptyDB(t *testing.T) {
	s := newTestStorage(t)

	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Errorf("LoadLite should not return error for empty db, got: %v", err)
	}
	if len(instData) != 0 {
		t.Errorf("Expected empty instances, got %d", len(instData))
	}
	if len(groupData) != 0 {
		t.Errorf("Expected empty groups, got %d", len(groupData))
	}
}
