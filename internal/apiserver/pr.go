package apiserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

)

const (
	prCacheTTL      = 60 * time.Second
	prRefreshPeriod = 60 * time.Second
)

// internalPREntry holds a cached PR result with its fetch timestamp.
// info == nil means "no PR found" and is also cached to avoid repeated gh calls.
type internalPREntry struct {
	info      *PRInfo
	fetchedAt time.Time
}

// internalPRCache is a simple TTL cache for PR info, keyed by session ID.
// Used when the TUI callback is unavailable (standalone web server mode) or
// as a fallback when the TUI cache hasn't fetched a given session yet.
type internalPRCache struct {
	mu      sync.RWMutex
	entries map[string]internalPREntry
}

func newInternalPRCache() *internalPRCache {
	return &internalPRCache{entries: make(map[string]internalPREntry)}
}

// get returns cached PR info if present and not expired.
// Returns (nil, true) if a nil result is cached (no PR found).
// Returns (nil, false) if the entry is missing or expired.
func (c *internalPRCache) get(sessionID string) (*PRInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[sessionID]
	if !ok || time.Since(e.fetchedAt) > prCacheTTL {
		return nil, false
	}
	return e.info, true
}

// set stores PR info (may be nil, meaning no PR found).
func (c *internalPRCache) set(sessionID string, info *PRInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[sessionID] = internalPREntry{info: info, fetchedAt: time.Now()}
}

// fetchPR runs `gh pr view` in the worktree directory and returns PR info.
// Returns nil if no PR exists or gh is unavailable.
func fetchPR(worktreePath string) *PRInfo {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return nil
	}

	cmd := exec.Command(ghPath, "pr", "view", "--json", "number,title,state,url,statusCheckRollup")
	cmd.Dir = worktreePath
	if host := ghHostFromPath(worktreePath); host != "" && host != "github.com" {
		cmd.Env = append(os.Environ(), "GH_HOST="+host)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil // no PR, or gh error (e.g. not in a git repo / no auth)
	}

	var raw struct {
		Number            int    `json:"number"`
		Title             string `json:"title"`
		State             string `json:"state"`
		URL               string `json:"url"`
		StatusCheckRollup []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil
	}

	pr := &PRInfo{
		Number: raw.Number,
		Title:  raw.Title,
		State:  raw.State,
		URL:    raw.URL,
	}
	for _, check := range raw.StatusCheckRollup {
		pr.HasChecks = true
		switch check.Status {
		case "COMPLETED":
			switch check.Conclusion {
			case "SUCCESS", "SKIPPED", "NEUTRAL":
				pr.ChecksPassed++
			default:
				pr.ChecksFailed++
			}
		default:
			pr.ChecksPending++
		}
	}
	return pr
}

// ghHostFromPath infers GH_HOST from the git remote URL in the given directory.
func ghHostFromPath(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	remote := strings.TrimSpace(string(out))
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}
	for _, scheme := range []string{"https://", "http://"} {
		if strings.HasPrefix(remote, scheme) {
			rest := strings.TrimPrefix(remote, scheme)
			return strings.SplitN(rest, "/", 2)[0]
		}
	}
	return ""
}

// runPRRefreshLoop performs an initial PR fetch then loops every prRefreshPeriod,
// refreshing PR info for all worktree sessions. Broadcasts sessions_changed after
// each refresh so web clients receive updated PR data.
func (s *APIServer) runPRRefreshLoop(ctx context.Context) {
	// Initial fetch after a brief delay so instances are loaded.
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Second):
	}
	s.refreshAllPRs()

	ticker := time.NewTicker(prRefreshPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshAllPRs()
		}
	}
}

// refreshAllPRs updates the internal PR cache for every worktree session
// and broadcasts sessions_changed so web clients pick up fresh PR data.
func (s *APIServer) refreshAllPRs() {
	instances := s.instances()
	refreshed := 0
	for _, inst := range instances {
		if inst.WorktreePath == "" {
			continue
		}
		pr := fetchPR(inst.WorktreePath)
		s.internalPR.set(inst.ID, pr)
		refreshed++
		slog.Debug("pr_refreshed", "session_id", inst.ID, "has_pr", pr != nil)
	}
	if refreshed > 0 {
		s.hub.broadcast <- WsMessage{Type: "sessions_changed"}
	}
}

// getPRInfoFor returns PR info for a session.
// It tries the TUI callback first (authoritative fast path when TUI is running),
// then falls back to the internal cache (standalone mode or sessions not visible in TUI).
func (s *APIServer) getPRInfoFor(sessionID string) *PRInfo {
	if s.getPRInfo != nil {
		if pr := s.getPRInfo(sessionID); pr != nil {
			return pr
		}
	}
	pr, ok := s.internalPR.get(sessionID)
	if ok {
		return pr
	}
	return nil
}

