package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusWatcher_DetectsActivityFile(t *testing.T) {
	// Create temp status directory
	statusDir := t.TempDir()

	// Create watcher
	sw, err := NewStatusWatcher(statusDir)
	require.NoError(t, err)
	defer sw.Close()

	sw.Start()

	// Small delay to ensure watcher is ready
	time.Sleep(50 * time.Millisecond)

	// Write an .active file (simulating tmux activity hook)
	instanceID := "test-instance-123"
	activeFile := filepath.Join(statusDir, instanceID+".active")
	content := `{"instance_id":"test-instance-123","event":"activity","timestamp":1234567890}`
	err = os.WriteFile(activeFile, []byte(content), 0644)
	require.NoError(t, err)

	// Should receive activity event
	select {
	case evt := <-sw.EventChannel():
		assert.Equal(t, "test-instance-123", evt.InstanceID)
		assert.Equal(t, "activity", evt.Event)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for activity event")
	}
}

func TestStatusWatcher_DetectsStopFile(t *testing.T) {
	// Create temp status directory
	statusDir := t.TempDir()

	// Create watcher
	sw, err := NewStatusWatcher(statusDir)
	require.NoError(t, err)
	defer sw.Close()

	sw.Start()

	// Small delay to ensure watcher is ready
	time.Sleep(50 * time.Millisecond)

	// Write a .stop file (existing functionality)
	instanceID := "test-instance-456"
	stopFile := filepath.Join(statusDir, instanceID+".stop")
	content := `{"instance_id":"test-instance-456","event":"stop","timestamp":1234567890}`
	err = os.WriteFile(stopFile, []byte(content), 0644)
	require.NoError(t, err)

	// Should receive stop event
	select {
	case evt := <-sw.EventChannel():
		assert.Equal(t, "test-instance-456", evt.InstanceID)
		assert.Equal(t, "stop", evt.Event)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for stop event")
	}
}

func TestStatusWatcher_IgnoresOtherFiles(t *testing.T) {
	// Create temp status directory
	statusDir := t.TempDir()

	// Create watcher
	sw, err := NewStatusWatcher(statusDir)
	require.NoError(t, err)
	defer sw.Close()

	sw.Start()

	// Small delay to ensure watcher is ready
	time.Sleep(50 * time.Millisecond)

	// Write a .txt file (should be ignored)
	txtFile := filepath.Join(statusDir, "test.txt")
	err = os.WriteFile(txtFile, []byte("some content"), 0644)
	require.NoError(t, err)

	// Write a .json file (should be ignored)
	jsonFile := filepath.Join(statusDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"event":"test"}`), 0644)
	require.NoError(t, err)

	// Should NOT receive any events
	select {
	case evt := <-sw.EventChannel():
		t.Fatalf("Should not have received event, got: %+v", evt)
	case <-time.After(500 * time.Millisecond):
		// Expected - no events should be received
	}
}
