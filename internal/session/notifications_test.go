package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationManager_Add(t *testing.T) {
	nm := NewNotificationManager(6)

	inst := &Instance{
		ID:     "abc123",
		Title:  "frontend",
		Status: StatusWaiting,
	}

	err := nm.Add(inst)
	require.NoError(t, err)

	entries := nm.GetEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "frontend", entries[0].Title)
	assert.Equal(t, "1", entries[0].AssignedKey)
}

func TestNotificationManager_NewestFirst(t *testing.T) {
	nm := NewNotificationManager(6)

	// Add three sessions with delays
	inst1 := &Instance{ID: "a", Title: "first", Status: StatusWaiting}
	nm.Add(inst1)
	time.Sleep(10 * time.Millisecond)

	inst2 := &Instance{ID: "b", Title: "second", Status: StatusWaiting}
	nm.Add(inst2)
	time.Sleep(10 * time.Millisecond)

	inst3 := &Instance{ID: "c", Title: "third", Status: StatusWaiting}
	nm.Add(inst3)

	entries := nm.GetEntries()
	assert.Len(t, entries, 3)
	// Newest should be at position [0] with key "1"
	assert.Equal(t, "third", entries[0].Title)
	assert.Equal(t, "1", entries[0].AssignedKey)
	assert.Equal(t, "second", entries[1].Title)
	assert.Equal(t, "2", entries[1].AssignedKey)
	assert.Equal(t, "first", entries[2].Title)
	assert.Equal(t, "3", entries[2].AssignedKey)
}

func TestNotificationManager_Remove(t *testing.T) {
	nm := NewNotificationManager(6)

	inst1 := &Instance{ID: "a", Title: "first", Status: StatusWaiting}
	inst2 := &Instance{ID: "b", Title: "second", Status: StatusWaiting}
	nm.Add(inst1)
	nm.Add(inst2)

	nm.Remove("b") // Remove newest

	entries := nm.GetEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "first", entries[0].Title)
	assert.Equal(t, "1", entries[0].AssignedKey) // Should shift to key "1"
}

func TestNotificationManager_MaxShown(t *testing.T) {
	nm := NewNotificationManager(3) // Max 3

	for i := 0; i < 5; i++ {
		inst := &Instance{ID: string(rune('a' + i)), Title: string(rune('A' + i)), Status: StatusWaiting}
		nm.Add(inst)
		time.Sleep(5 * time.Millisecond)
	}

	entries := nm.GetEntries()
	assert.Len(t, entries, 3) // Only 3 shown
	// Newest 3 should be shown
	assert.Equal(t, "E", entries[0].Title) // newest
	assert.Equal(t, "D", entries[1].Title)
	assert.Equal(t, "C", entries[2].Title)
}

func TestNotificationManager_FormatBar(t *testing.T) {
	nm := NewNotificationManager(6)

	// Empty bar
	assert.Equal(t, "", nm.FormatBar())

	// One session
	nm.Add(&Instance{ID: "a", Title: "frontend", Status: StatusWaiting})
	bar := nm.FormatBar()
	assert.Contains(t, bar, "[1]")
	assert.Contains(t, bar, "frontend")

	// Two sessions
	nm.Add(&Instance{ID: "b", Title: "api", Status: StatusWaiting})
	bar = nm.FormatBar()
	assert.Contains(t, bar, "[1]")
	assert.Contains(t, bar, "[2]")
}

func TestNotificationManager_FullTitles(t *testing.T) {
	nm := NewNotificationManager(6)

	// Add 6 sessions with long names
	for i := 0; i < 6; i++ {
		inst := &Instance{
			ID:     string(rune('a' + i)),
			Title:  "verylongsessionname" + string(rune('0'+i)),
			Status: StatusWaiting,
		}
		nm.Add(inst)
	}

	bar := nm.FormatBar()
	// Full titles should be shown (no truncation)
	// Each title is ~20 chars, bar should contain all of them
	assert.Contains(t, bar, "verylongsessionname5") // newest
	assert.Contains(t, bar, "verylongsessionname0") // oldest
}

func TestNotificationManager_GetSessionByKey(t *testing.T) {
	nm := NewNotificationManager(6)

	inst1 := &Instance{ID: "a", Title: "first", Status: StatusWaiting}
	inst2 := &Instance{ID: "b", Title: "second", Status: StatusWaiting}
	nm.Add(inst1)
	nm.Add(inst2)

	// Key "1" should return newest (second)
	entry := nm.GetSessionByKey("1")
	require.NotNil(t, entry)
	assert.Equal(t, "b", entry.SessionID)

	// Key "2" should return first
	entry = nm.GetSessionByKey("2")
	require.NotNil(t, entry)
	assert.Equal(t, "a", entry.SessionID)

	// Key "3" should return nil
	entry = nm.GetSessionByKey("3")
	assert.Nil(t, entry)
}

func TestNotificationManager_Count(t *testing.T) {
	nm := NewNotificationManager(6)

	assert.Equal(t, 0, nm.Count())

	nm.Add(&Instance{ID: "a", Title: "first", Status: StatusWaiting})
	assert.Equal(t, 1, nm.Count())

	nm.Add(&Instance{ID: "b", Title: "second", Status: StatusWaiting})
	assert.Equal(t, 2, nm.Count())

	nm.Remove("a")
	assert.Equal(t, 1, nm.Count())
}

func TestNotificationManager_Clear(t *testing.T) {
	nm := NewNotificationManager(6)

	nm.Add(&Instance{ID: "a", Title: "first", Status: StatusWaiting})
	nm.Add(&Instance{ID: "b", Title: "second", Status: StatusWaiting})
	assert.Equal(t, 2, nm.Count())

	nm.Clear()
	assert.Equal(t, 0, nm.Count())
	assert.Empty(t, nm.GetEntries())
}

func TestNotificationManager_Has(t *testing.T) {
	nm := NewNotificationManager(6)

	nm.Add(&Instance{ID: "a", Title: "first", Status: StatusWaiting})

	assert.True(t, nm.Has("a"))
	assert.False(t, nm.Has("b"))
}

func TestNotificationManager_DuplicateAdd(t *testing.T) {
	nm := NewNotificationManager(6)

	inst := &Instance{ID: "a", Title: "first", Status: StatusWaiting}
	nm.Add(inst)
	nm.Add(inst) // Add same instance again

	// Should only have one entry
	assert.Equal(t, 1, nm.Count())
}

func TestNotificationManager_SyncFromInstances(t *testing.T) {
	nm := NewNotificationManager(6)

	// Initial add
	nm.Add(&Instance{ID: "a", Title: "first", Status: StatusWaiting})

	instances := []*Instance{
		{ID: "a", Title: "first", Status: StatusWaiting},  // Still waiting
		{ID: "b", Title: "second", Status: StatusWaiting}, // New waiting
		{ID: "c", Title: "third", Status: StatusIdle},     // Not waiting
	}

	added, removed := nm.SyncFromInstances(instances, "")

	assert.Contains(t, added, "b")
	assert.Empty(t, removed)
	assert.Equal(t, 2, nm.Count())
	assert.True(t, nm.Has("a"))
	assert.True(t, nm.Has("b"))
	assert.False(t, nm.Has("c"))
}

func TestNotificationManager_SyncFromInstances_RemovesNonWaiting(t *testing.T) {
	nm := NewNotificationManager(6)

	nm.Add(&Instance{ID: "a", Title: "first", Status: StatusWaiting})
	nm.Add(&Instance{ID: "b", Title: "second", Status: StatusWaiting})

	// "a" is no longer waiting (became idle)
	instances := []*Instance{
		{ID: "a", Title: "first", Status: StatusIdle},
		{ID: "b", Title: "second", Status: StatusWaiting},
	}

	added, removed := nm.SyncFromInstances(instances, "")

	assert.Empty(t, added)
	assert.Contains(t, removed, "a")
	assert.Equal(t, 1, nm.Count())
	assert.False(t, nm.Has("a"))
	assert.True(t, nm.Has("b"))
}

func TestNotificationManager_SyncFromInstances_ExcludesCurrentSession(t *testing.T) {
	nm := NewNotificationManager(6)

	instances := []*Instance{
		{ID: "current", Title: "current-session", Status: StatusWaiting},
		{ID: "other", Title: "other-session", Status: StatusWaiting},
	}

	// Sync with "current" as the current session - it should be excluded
	added, _ := nm.SyncFromInstances(instances, "current")

	assert.Contains(t, added, "other")
	assert.NotContains(t, added, "current")
	assert.Equal(t, 1, nm.Count())
	assert.False(t, nm.Has("current"))
	assert.True(t, nm.Has("other"))
}

func TestNotificationManager_DefaultMaxShown(t *testing.T) {
	nm := NewNotificationManager(0) // Invalid value should default to 6

	for i := 0; i < 10; i++ {
		nm.Add(&Instance{ID: string(rune('a' + i)), Title: string(rune('A' + i)), Status: StatusWaiting})
	}

	assert.Equal(t, 6, nm.Count())
}
