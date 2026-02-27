package ui

import (
	"sync"
	"time"
)

// TTLs for each cache type, matching the original per-cache constants.
const (
	previewCacheTTL      = 2 * time.Second
	worktreeDirtyCacheTTL = 10 * time.Second
	worktreeRemoteCacheTTL = 5 * time.Minute
	prCacheTTL            = 60 * time.Second
)

// cacheEntry holds a cached value with its TTL metadata.
type cacheEntry struct {
	value    any
	cachedAt time.Time
	ttl      time.Duration
}

func (e *cacheEntry) isExpired() bool {
	return time.Since(e.cachedAt) > e.ttl
}

// UICache consolidates the UI layer's per-session caches into a single
// thread-safe store with TTL-based expiry.
//
// Replaces four independent maps in Home:
//   - previewCache / previewCacheTime / previewCacheMu  (2s TTL)
//   - worktreeDirtyCache / worktreeDirtyCacheTs / worktreeDirtyMu  (10s TTL)
//   - worktreeRemoteCache / worktreeRemoteCacheTs / worktreeRemoteMu  (5m TTL)
//   - prCache / prCacheTs / prCacheMu  (60s TTL)
type UICache struct {
	mu      sync.RWMutex
	entries map[string]map[string]cacheEntry // [sessionID][key] -> entry
}

func newUICache() *UICache {
	return &UICache{
		entries: make(map[string]map[string]cacheEntry),
	}
}

func (c *UICache) get(sessionID, key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return nil, false
	}
	e, ok := bucket[key]
	if !ok || e.isExpired() {
		return nil, false
	}
	return e.value, true
}

// getWithTime returns the value and its cache timestamp regardless of expiry.
// Used when callers need to inspect the timestamp directly (e.g. to stamp
// duplicate-fetch prevention without a fresh value).
func (c *UICache) getWithTime(sessionID, key string) (any, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return nil, time.Time{}, false
	}
	e, ok := bucket[key]
	if !ok {
		return nil, time.Time{}, false
	}
	return e.value, e.cachedAt, true
}

func (c *UICache) set(sessionID, key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries[sessionID] == nil {
		c.entries[sessionID] = make(map[string]cacheEntry)
	}
	c.entries[sessionID][key] = cacheEntry{value: value, cachedAt: time.Now(), ttl: ttl}
}

// touch stamps a new cachedAt for key without changing the value. Used to
// prevent duplicate background fetches before the real result arrives.
func (c *UICache) touch(sessionID, key string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries[sessionID] == nil {
		c.entries[sessionID] = make(map[string]cacheEntry)
	}
	existing := c.entries[sessionID][key]
	c.entries[sessionID][key] = cacheEntry{value: existing.value, cachedAt: time.Now(), ttl: ttl}
}

// InvalidateSession removes all cache entries for a session.
func (c *UICache) InvalidateSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, sessionID)
}

// InvalidateAll clears every cache entry.
func (c *UICache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]map[string]cacheEntry)
}

// ---------------------------------------------------------------------------
// Typed accessors — Preview (2s TTL)
// ---------------------------------------------------------------------------

const cacheKeyPreview = "preview"

// GetPreview returns the cached terminal preview content for the session,
// along with its cache timestamp. The bool is false if no entry exists at all
// (expired or never cached). Callers that only need the content should use
// the bool; callers that need the timestamp for TTL comparisons should
// inspect cachedAt directly.
func (c *UICache) GetPreview(sessionID string) (content string, cachedAt time.Time, ok bool) {
	v, t, exists := c.getWithTime(sessionID, cacheKeyPreview)
	if !exists {
		return "", time.Time{}, false
	}
	s, _ := v.(string)
	return s, t, true
}

// SetPreview stores the preview content for the session.
func (c *UICache) SetPreview(sessionID, content string) {
	c.set(sessionID, cacheKeyPreview, content, previewCacheTTL)
}

// TouchPreview resets the cachedAt timestamp for the preview entry without
// changing the content. Used to prevent duplicate fetch goroutines.
func (c *UICache) TouchPreview(sessionID string) {
	c.touch(sessionID, cacheKeyPreview, previewCacheTTL)
}

// ---------------------------------------------------------------------------
// Typed accessors — Worktree dirty status (10s TTL)
// ---------------------------------------------------------------------------

const cacheKeyWorktreeDirty = "wtDirty"

// GetWorktreeDirty returns the cached dirty flag and whether the entry exists.
// If the entry is expired, ok is false.
func (c *UICache) GetWorktreeDirty(sessionID string) (isDirty bool, ok bool) {
	v, exists := c.get(sessionID, cacheKeyWorktreeDirty)
	if !exists {
		return false, false
	}
	b, _ := v.(bool)
	return b, true
}

// HasWorktreeDirtyEntry returns true if a dirty-status entry exists for the
// session (whether or not it is still valid). This is used to distinguish
// "never fetched" from "fetched and clean".
func (c *UICache) HasWorktreeDirtyEntry(sessionID string) (isDirty bool, exists bool, expired bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return false, false, false
	}
	e, ok := bucket[cacheKeyWorktreeDirty]
	if !ok {
		return false, false, false
	}
	b, _ := e.value.(bool)
	return b, true, e.isExpired()
}

// SetWorktreeDirty stores the dirty flag for the session.
func (c *UICache) SetWorktreeDirty(sessionID string, isDirty bool) {
	c.set(sessionID, cacheKeyWorktreeDirty, isDirty, worktreeDirtyCacheTTL)
}

// TouchWorktreeDirty stamps a new cachedAt without changing the dirty value,
// used to prevent duplicate background git checks.
func (c *UICache) TouchWorktreeDirty(sessionID string) {
	c.touch(sessionID, cacheKeyWorktreeDirty, worktreeDirtyCacheTTL)
}

// GetWorktreeDirtyCachedAt returns the cache timestamp for the dirty entry
// (zero time if no entry exists). Used for TTL-deduplication logic.
func (c *UICache) GetWorktreeDirtyCachedAt(sessionID string) (cachedAt time.Time, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return time.Time{}, false
	}
	e, ok := bucket[cacheKeyWorktreeDirty]
	if !ok {
		return time.Time{}, false
	}
	return e.cachedAt, true
}

// InvalidateWorktreeDirty removes only the dirty-status entry for a session.
func (c *UICache) InvalidateWorktreeDirty(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if bucket, ok := c.entries[sessionID]; ok {
		delete(bucket, cacheKeyWorktreeDirty)
	}
}

// ---------------------------------------------------------------------------
// Typed accessors — Worktree remote URL (5m TTL)
// ---------------------------------------------------------------------------

const cacheKeyWorktreeRemote = "wtRemote"

// GetWorktreeRemote returns the cached remote URL. ok is false if the entry
// is absent or expired.
func (c *UICache) GetWorktreeRemote(sessionID string) (remoteURL string, ok bool) {
	v, exists := c.get(sessionID, cacheKeyWorktreeRemote)
	if !exists {
		return "", false
	}
	s, _ := v.(string)
	return s, true
}

// HasWorktreeRemoteEntry returns the remote URL and whether any entry (even
// expired) exists. Used to display cached data while a refresh is pending.
func (c *UICache) HasWorktreeRemoteEntry(sessionID string) (remoteURL string, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return "", false
	}
	e, ok := bucket[cacheKeyWorktreeRemote]
	if !ok {
		return "", false
	}
	s, _ := e.value.(string)
	return s, true
}

// SetWorktreeRemote stores the remote URL for the session.
func (c *UICache) SetWorktreeRemote(sessionID, remoteURL string) {
	c.set(sessionID, cacheKeyWorktreeRemote, remoteURL, worktreeRemoteCacheTTL)
}

// GetWorktreeRemoteCachedAt returns the cache timestamp for the remote URL
// entry (zero time if no entry exists). Used for TTL-deduplication logic.
func (c *UICache) GetWorktreeRemoteCachedAt(sessionID string) (cachedAt time.Time, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return time.Time{}, false
	}
	e, ok := bucket[cacheKeyWorktreeRemote]
	if !ok {
		return time.Time{}, false
	}
	return e.cachedAt, true
}

// TouchWorktreeRemote stamps a new cachedAt to prevent duplicate fetches.
func (c *UICache) TouchWorktreeRemote(sessionID string) {
	c.touch(sessionID, cacheKeyWorktreeRemote, worktreeRemoteCacheTTL)
}

// ---------------------------------------------------------------------------
// Typed accessors — PR info (60s TTL)
// ---------------------------------------------------------------------------

const cacheKeyPR = "pr"

// GetPR returns the cached prCacheEntry. The bool is false if the entry is
// absent or expired. A nil *prCacheEntry with ok=true means "fetched but no
// PR found".
func (c *UICache) GetPR(sessionID string) (pr *prCacheEntry, ok bool) {
	v, exists := c.get(sessionID, cacheKeyPR)
	if !exists {
		return nil, false
	}
	// v may be a typed nil (*prCacheEntry)(nil) stored as any.
	pr, _ = v.(*prCacheEntry)
	return pr, true
}

// HasPREntry returns the cached prCacheEntry and whether any entry (even
// expired) exists. Used to gate fetch-deduplication by timestamp.
func (c *UICache) HasPREntry(sessionID string) (pr *prCacheEntry, cachedAt time.Time, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.entries[sessionID]
	if !ok {
		return nil, time.Time{}, false
	}
	e, ok := bucket[cacheKeyPR]
	if !ok {
		return nil, time.Time{}, false
	}
	p, _ := e.value.(*prCacheEntry)
	return p, e.cachedAt, true
}

// SetPR stores the PR info entry for the session (nil is a valid value meaning
// "no PR found").
func (c *UICache) SetPR(sessionID string, pr *prCacheEntry) {
	c.set(sessionID, cacheKeyPR, pr, prCacheTTL)
}

// TouchPR stamps a new cachedAt to prevent duplicate fetch goroutines.
func (c *UICache) TouchPR(sessionID string) {
	c.touch(sessionID, cacheKeyPR, prCacheTTL)
}

// InvalidatePRTimestamp removes only the PR timestamp for a session so the
// next poll will re-fetch even if the value is present. This is accomplished
// by deleting just the PR entry.
func (c *UICache) InvalidatePRTimestamp(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if bucket, ok := c.entries[sessionID]; ok {
		delete(bucket, cacheKeyPR)
	}
}
