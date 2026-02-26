# Preview Pane: Remote URL + Tilde Paths

**Date:** 2026-02-25

## Summary

Improve the worktree session preview pane by:
1. Adding a `Remote:` line showing the git origin URL (normalized)
2. Removing the redundant `Repo:` line (implied by worktree path)
3. Moving PR + Checks to the top of the worktree section
4. Tilde-compressing all displayed paths (`/Users/username/...` → `~/...`)

## Final Layout

```
PR:      #42 open · Fix the thing
         https://github.com/...
Checks:  ✓ 12 passed
Branch:  my-feature
Remote:  github: mnicholson/hangar
Path:    ~/.worktrees/repo-in-preview,...
Status:  clean
Finish:  W finish + cleanup
```

## Changes

### 1. `shortenPath` helper (internal/ui/home.go)

New function that tilde-compresses before passing to `truncatePath`:

```go
func shortenPath(path string, maxLen int) string {
    if home, err := os.UserHomeDir(); err == nil {
        if strings.HasPrefix(path, home+"/") {
            path = "~/" + path[len(home)+1:]
        } else if path == home {
            path = "~"
        }
    }
    return truncatePath(path, maxLen)
}
```

Applied to: `selected.ProjectPath` (📁 line) and `selected.WorktreePath`.

### 2. Remove `Repo:` block

Delete the `WorktreeRepoRoot` display block (lines 7526–7531 in home.go).

### 3. Add `Remote:` line with lazy cache

Add to home model:
- `worktreeRemoteCache map[string]string`   — session ID → normalized URL
- `worktreeRemoteCacheMu sync.Mutex`
- `worktreeRemoteCacheTs map[string]time.Time` — for TTL

Fetch via `git -C <WorktreePath> config --get remote.origin.url`. TTL: 5 minutes.
Triggered in the preview debounce handler (same as dirty status).

Placement: between Branch and Path (replacing Repo slot).

### 4. Remote URL normalization

```
git@github.com:user/repo.git       → github: user/repo
git@ghe.spotify.net:user/repo.git  → ghe: user/repo
https://github.com/user/repo.git   → github: user/repo
https://ghe.spotify.net/user/repo  → ghe: user/repo
<anything else>                    → displayed as-is (truncated)
```

### 5. Reorder worktree section

New order: PR → Checks → Branch → Remote → Path → Status → Finish hint.
(Previously: Branch → PR → Checks → Repo → Path → Status → Finish)
