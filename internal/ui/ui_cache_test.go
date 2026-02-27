package ui

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestUICache_BasicOperations(t *testing.T) {
	c := newUICache()

	// Set and get preview
	c.SetPreview("sess-1", "content")
	v, _, ok := c.GetPreview("sess-1")
	if !ok || v != "content" {
		t.Errorf("GetPreview = %q, %v; want content, true", v, ok)
	}

	// Missing key
	_, _, ok = c.GetPreview("sess-2")
	if ok {
		t.Error("expected miss for nonexistent session")
	}

	// Invalidation
	c.InvalidateSession("sess-1")
	_, _, ok = c.GetPreview("sess-1")
	if ok {
		t.Error("expected miss after InvalidateSession")
	}
}

func TestUICache_WorktreeDirty(t *testing.T) {
	c := newUICache()

	c.SetWorktreeDirty("sess-1", true)
	isDirty, ok := c.GetWorktreeDirty("sess-1")
	if !ok || !isDirty {
		t.Errorf("GetWorktreeDirty = %v, %v; want true, true", isDirty, ok)
	}

	c.SetWorktreeDirty("sess-1", false)
	isDirty, ok = c.GetWorktreeDirty("sess-1")
	if !ok || isDirty {
		t.Errorf("GetWorktreeDirty after set false = %v, %v; want false, true", isDirty, ok)
	}

	// Missing
	_, ok = c.GetWorktreeDirty("sess-2")
	if ok {
		t.Error("expected miss for nonexistent session")
	}

	// InvalidateWorktreeDirty removes only dirty entry
	c.InvalidateWorktreeDirty("sess-1")
	_, ok = c.GetWorktreeDirty("sess-1")
	if ok {
		t.Error("expected miss after InvalidateWorktreeDirty")
	}
}

func TestUICache_WorktreeRemote(t *testing.T) {
	c := newUICache()

	c.SetWorktreeRemote("sess-1", "https://github.com/org/repo")
	url, ok := c.GetWorktreeRemote("sess-1")
	if !ok || url != "https://github.com/org/repo" {
		t.Errorf("GetWorktreeRemote = %q, %v; want url, true", url, ok)
	}

	_, ok = c.GetWorktreeRemote("sess-2")
	if ok {
		t.Error("expected miss for nonexistent session")
	}
}

func TestUICache_PR(t *testing.T) {
	c := newUICache()

	entry := &prCacheEntry{Number: 42, State: "OPEN"}
	c.SetPR("sess-1", entry)

	pr, ok := c.GetPR("sess-1")
	if !ok || pr == nil || pr.Number != 42 {
		t.Errorf("GetPR = %v, %v; want entry with Number=42, true", pr, ok)
	}

	// nil entry (no PR found)
	c.SetPR("sess-2", nil)
	pr, ok = c.GetPR("sess-2")
	if !ok {
		t.Error("GetPR for nil entry should return ok=true (key exists)")
	}
	if pr != nil {
		t.Errorf("GetPR for nil entry should return nil pr, got %v", pr)
	}

	// Missing
	_, ok = c.GetPR("sess-3")
	if ok {
		t.Error("expected miss for nonexistent session")
	}

	// HasPREntry preserves expired entries
	pr2, _, exists := c.HasPREntry("sess-1")
	if !exists || pr2 == nil || pr2.Number != 42 {
		t.Errorf("HasPREntry = %v, %v; want entry, true", pr2, exists)
	}
}

func TestUICache_InvalidateAll(t *testing.T) {
	c := newUICache()

	c.SetPreview("sess-1", "content1")
	c.SetPreview("sess-2", "content2")
	c.SetWorktreeDirty("sess-1", true)

	c.InvalidateAll()

	_, _, ok1 := c.GetPreview("sess-1")
	_, _, ok2 := c.GetPreview("sess-2")
	_, ok3 := c.GetWorktreeDirty("sess-1")
	if ok1 || ok2 || ok3 {
		t.Errorf("expected all entries cleared after InvalidateAll, got ok1=%v ok2=%v ok3=%v", ok1, ok2, ok3)
	}
}

func TestUICache_TTLExpiry(t *testing.T) {
	c := newUICache()

	// Store with a very short custom TTL via the lower-level set method
	c.mu.Lock()
	if c.entries["sess-1"] == nil {
		c.entries["sess-1"] = make(map[string]cacheEntry)
	}
	c.entries["sess-1"][cacheKeyWorktreeDirty] = cacheEntry{
		value:    true,
		cachedAt: time.Now().Add(-10 * time.Second), // 10s ago
		ttl:      1 * time.Second,                   // 1s TTL — already expired
	}
	c.mu.Unlock()

	// GetWorktreeDirty uses the expiry-checking get() method
	_, ok := c.GetWorktreeDirty("sess-1")
	if ok {
		t.Error("expected miss for expired entry via GetWorktreeDirty")
	}

	// GetPreview returns stale content (callers show last-known content while refresh
	// is in-flight); expiry is checked via the cachedAt timestamp, not ok bool.
	c.mu.Lock()
	if c.entries["sess-2"] == nil {
		c.entries["sess-2"] = make(map[string]cacheEntry)
	}
	c.entries["sess-2"][cacheKeyPreview] = cacheEntry{
		value:    "stale content",
		cachedAt: time.Now().Add(-10 * time.Second), // 10s ago
		ttl:      1 * time.Second,                   // 1s TTL — already expired
	}
	c.mu.Unlock()

	content, cachedAt, okPreview := c.GetPreview("sess-2")
	if !okPreview || content != "stale content" {
		t.Errorf("GetPreview should return stale content; got content=%q ok=%v", content, okPreview)
	}
	if time.Since(cachedAt) < 5*time.Second {
		t.Errorf("cachedAt should be old (>5s ago), got %v ago", time.Since(cachedAt))
	}
}

func TestUICache_Touch(t *testing.T) {
	c := newUICache()

	c.SetWorktreeDirty("sess-1", true)
	cachedAt1, _ := c.GetWorktreeDirtyCachedAt("sess-1")

	// Brief sleep to ensure timestamp changes
	time.Sleep(2 * time.Millisecond)
	c.TouchWorktreeDirty("sess-1")
	cachedAt2, _ := c.GetWorktreeDirtyCachedAt("sess-1")

	if !cachedAt2.After(cachedAt1) {
		t.Errorf("TouchWorktreeDirty did not advance cachedAt: %v -> %v", cachedAt1, cachedAt2)
	}

	// Value should be preserved
	isDirty, ok := c.GetWorktreeDirty("sess-1")
	if !ok || !isDirty {
		t.Errorf("GetWorktreeDirty after Touch = %v, %v; want true, true", isDirty, ok)
	}
}

func TestUICache_InvalidatePRTimestamp(t *testing.T) {
	c := newUICache()

	entry := &prCacheEntry{Number: 99, State: "MERGED"}
	c.SetPR("sess-1", entry)

	// HasPREntry sees the entry
	_, _, exists := c.HasPREntry("sess-1")
	if !exists {
		t.Fatal("expected PR entry to exist")
	}

	// Invalidating the timestamp removes the entry entirely
	c.InvalidatePRTimestamp("sess-1")
	_, _, existsAfter := c.HasPREntry("sess-1")
	if existsAfter {
		t.Error("expected PR entry to be gone after InvalidatePRTimestamp")
	}
}

func TestUICache_ConcurrentAccess(t *testing.T) {
	c := newUICache()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("sess-%d", n%5)
			c.SetPreview(id, "content")
			c.GetPreview(id)
			c.SetWorktreeDirty(id, n%2 == 0)
			c.GetWorktreeDirty(id)
			c.InvalidateSession(id)
		}(i)
	}
	wg.Wait()
}
