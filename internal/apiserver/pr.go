package apiserver

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/sjoeboo/hangar/internal/pr"
)

// fetchPR runs `gh pr view` in the worktree directory and returns PR info.
// Returns nil if no PR exists or gh is unavailable.
// This is the legacy fallback used when prManager is nil (standalone mode).
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

	info := &PRInfo{
		Number: raw.Number,
		Title:  raw.Title,
		State:  raw.State,
		URL:    raw.URL,
	}
	for _, check := range raw.StatusCheckRollup {
		info.HasChecks = true
		switch check.Status {
		case "COMPLETED":
			switch check.Conclusion {
			case "SUCCESS", "SKIPPED", "NEUTRAL":
				info.ChecksPassed++
			default:
				info.ChecksFailed++
			}
		default:
			info.ChecksPending++
		}
	}
	return info
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

// prInfoFromPR converts a *pr.PR to a *PRInfo for use in session responses.
func prInfoFromPR(p *pr.PR) *PRInfo {
	if p == nil {
		return nil
	}
	return &PRInfo{
		Number:        p.Number,
		Title:         p.Title,
		State:         p.State,
		URL:           p.URL,
		ChecksPassed:  p.ChecksPassed,
		ChecksFailed:  p.ChecksFailed,
		ChecksPending: p.ChecksPending,
		HasChecks:     p.HasChecks,
	}
}

// getPRInfoFor returns PR info for a session.
// It tries the pr.Manager first (authoritative when available), then falls back
// to the TUI callback, then to a direct gh fetch.
func (s *APIServer) getPRInfoFor(sessionID string) *PRInfo {
	// Try pr.Manager first
	if s.prManager != nil {
		if p, exists := s.prManager.GetSessionPR(sessionID); exists && p != nil {
			return prInfoFromPR(p)
		}
	}
	// TUI callback (legacy)
	if s.getPRInfo != nil {
		if info := s.getPRInfo(sessionID); info != nil {
			return info
		}
	}
	return nil
}
