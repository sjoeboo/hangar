# De-Spotify Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove all Spotify-specific hardcoding so Hangar can be open-sourced at `github.com/sjoeboo/hangar`.

**Architecture:** Three runtime changes — (1) configurable host→label mapping for the preview pane remote URL display, (2) automatic `GH_HOST` injection so `gh` CLI works across GitHub.com and GHE, (3) update repo constants and feedback URLs. Shell scripts updated in the same PR.

**Tech Stack:** Go, TOML config, `exec.Command`, `strings` package, existing test patterns in `styles_test.go`.

---

## Task 1: Add `RemoteLabels` to `UserConfig`

**Files:**
- Modify: `internal/session/userconfig.go:100–105` (struct field)
- Modify: `internal/session/userconfig.go:1494–1497` (config template, just before closing backtick)

**Step 1: Add struct field**

In `userconfig.go`, add before the `Tmux TmuxSettings` line (around line 103):

```go
// RemoteLabels maps git remote hostnames to short display labels shown in
// the preview pane. github.com → "github" is always built-in.
// Example: "ghe.mycompany.com" = "github"
RemoteLabels map[string]string `toml:"remote_labels"`
```

**Step 2: Add to config template**

Find the closing backtick of `exampleConfig` (around line 1497). Insert before it:

```
# ============================================================================
# Remote Host Display Labels
# ============================================================================
# Maps git remote hostnames to short labels shown in the preview pane.
# github.com → "github" is built-in. Add entries for self-hosted GHE or
# GitLab instances.
#
# [remote_labels]
# "ghe.mycompany.com" = "github"
# "gitlab.mycompany.com" = "gitlab"
```

**Step 3: Build to confirm no compilation errors**

```bash
go build ./...
```

Expected: no output (success).

**Step 4: Commit**

```bash
git add internal/session/userconfig.go
git commit -m "feat(config): add RemoteLabels map for custom git host display labels"
```

---

## Task 2: Update `normalizeRemoteURL` — signature and logic

**Files:**
- Modify: `internal/ui/home.go:6618–6645`

**Step 1: Replace the function**

Find `func normalizeRemoteURL(u string) string` (around line 6625) and replace the entire function:

```go
// normalizeRemoteURL converts a git remote URL to a short human-readable form.
//
//	git@github.com:user/repo.git         -> "github: user/repo"
//	https://github.com/user/repo.git     -> "github: user/repo"
//	git@HOST:user/repo.git               -> "<label>: user/repo"  (if HOST in extraLabels)
//	anything else                        -> returned unchanged
//
// extraLabels maps hostname → label (e.g. "ghe.mycompany.com" → "github").
// github.com is always built-in and cannot be overridden.
func normalizeRemoteURL(u string, extraLabels map[string]string) string {
	// Built-in: github.com (SSH + HTTPS)
	if rest, ok := strings.CutPrefix(u, "git@github.com:"); ok {
		return "github: " + strings.TrimSuffix(rest, ".git")
	}
	for _, prefix := range []string{"https://github.com/", "http://github.com/"} {
		if rest, ok := strings.CutPrefix(u, prefix); ok {
			return "github: " + strings.TrimSuffix(rest, ".git")
		}
	}
	// Config-driven labels
	if len(extraLabels) > 0 {
		host := extractRemoteHost(u)
		if label, ok := extraLabels[host]; ok && host != "" {
			var path string
			if rest, ok2 := strings.CutPrefix(u, "git@"+host+":"); ok2 {
				path = strings.TrimSuffix(rest, ".git")
			} else {
				for _, scheme := range []string{"https://", "http://"} {
					if rest, ok2 := strings.CutPrefix(u, scheme+host+"/"); ok2 {
						path = strings.TrimSuffix(rest, ".git")
						break
					}
				}
			}
			if path != "" {
				return label + ": " + path
			}
		}
	}
	return u
}
```

**Step 2: Build**

```bash
go build ./...
```

Expected: compilation error — `update_handlers.go` still calls the old one-argument form. That is expected; fix in the next task.

---

## Task 3: Add `extractRemoteHost` helper + tests

**Files:**
- Modify: `internal/ui/home.go` (add near `normalizeRemoteURL`, around line 6646)
- Modify: `internal/ui/styles_test.go`

**Step 1: Add helper immediately after `normalizeRemoteURL`**

```go
// extractRemoteHost parses a git remote URL and returns just the hostname.
//   git@github.com:user/repo.git  → "github.com"
//   https://github.com/user/repo  → "github.com"
func extractRemoteHost(u string) string {
	if after, ok := strings.CutPrefix(u, "git@"); ok {
		host, _, _ := strings.Cut(after, ":")
		return host
	}
	for _, scheme := range []string{"https://", "http://"} {
		if after, ok := strings.CutPrefix(u, scheme); ok {
			host, _, _ := strings.Cut(after, "/")
			return host
		}
	}
	return ""
}
```

**Step 2: Write tests in `styles_test.go`**

Replace the existing `TestNormalizeRemoteURL` entirely and add `TestExtractRemoteHost`:

```go
func TestExtractRemoteHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:user/repo.git", "github.com"},
		{"git@ghe.mycompany.com:user/repo.git", "ghe.mycompany.com"},
		{"https://github.com/user/repo.git", "github.com"},
		{"https://ghe.mycompany.com/user/repo", "ghe.mycompany.com"},
		{"http://gitlab.company.com/user/repo", "gitlab.company.com"},
		{"", ""},
		{"not-a-url", ""},
	}
	for _, tt := range tests {
		got := extractRemoteHost(tt.input)
		if got != tt.want {
			t.Errorf("extractRemoteHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeRemoteURL(t *testing.T) {
	customLabels := map[string]string{"ghe.mycompany.com": "github"}

	tests := []struct {
		input  string
		labels map[string]string
		want   string
	}{
		// Built-in github.com — always works regardless of labels map
		{"git@github.com:mnicholson/hangar.git", nil, "github: mnicholson/hangar"},
		{"https://github.com/mnicholson/hangar.git", nil, "github: mnicholson/hangar"},
		{"https://github.com/mnicholson/hangar", nil, "github: mnicholson/hangar"},
		// Custom label via config (SSH)
		{"git@ghe.mycompany.com:squad/service.git", customLabels, "github: squad/service"},
		// Custom label via config (HTTPS, with and without .git)
		{"https://ghe.mycompany.com/squad/service.git", customLabels, "github: squad/service"},
		{"https://ghe.mycompany.com/squad/service", customLabels, "github: squad/service"},
		// No label configured for host → raw URL unchanged
		{"git@ghe.mycompany.com:squad/service.git", nil, "git@ghe.mycompany.com:squad/service.git"},
		{"git@gitlab.com:user/repo.git", nil, "git@gitlab.com:user/repo.git"},
		// Empty input
		{"", nil, ""},
	}
	for _, tt := range tests {
		got := normalizeRemoteURL(tt.input, tt.labels)
		if got != tt.want {
			t.Errorf("normalizeRemoteURL(%q, %v) = %q, want %q", tt.input, tt.labels, got, tt.want)
		}
	}
}
```

**Step 3: Run only the affected tests (they will fail until callers are updated)**

```bash
go test ./internal/ui/... -run 'TestNormalizeRemoteURL|TestExtractRemoteHost' -v 2>&1 | head -30
```

Expected: `TestExtractRemoteHost` passes; `TestNormalizeRemoteURL` may fail on build until caller is updated in Task 4.

---

## Task 4: Fix `update_handlers.go` caller

**Files:**
- Modify: `internal/ui/update_handlers.go:794`

**Step 1: Update the call**

Find (around line 794):
```go
return worktreeRemoteCheckMsg{sessionID: sid, remoteURL: normalizeRemoteURL(url), err: err}
```

Replace with:
```go
return worktreeRemoteCheckMsg{sessionID: sid, remoteURL: normalizeRemoteURL(url, h.cfg.RemoteLabels), err: err}
```

**Step 2: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... -run 'TestNormalizeRemoteURL|TestExtractRemoteHost' -v
```

Expected: both tests pass.

**Step 3: Run full test suite**

```bash
go test ./... 2>&1 | grep -v '^ok' | grep -v '^?'
```

Expected: only the two known pre-existing failures (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`).

**Step 4: Commit**

```bash
git add internal/ui/home.go internal/ui/styles_test.go internal/ui/update_handlers.go
git commit -m "feat(ui): configurable remote host display labels via [remote_labels] config"
```

---

## Task 5: Add `ghHostFromDir` helper and update `fetchPRInfo`

**Files:**
- Modify: `internal/ui/home.go` — add helper + update `fetchPRInfo` (line 6573)

**Step 1: Add `ghHostFromDir` immediately after `extractRemoteHost`**

```go
// ghHostFromDir returns the git remote hostname for the origin remote in dir,
// or "" if it cannot be determined. Used to set GH_HOST for gh CLI invocations.
func ghHostFromDir(dir string) string {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return extractRemoteHost(strings.TrimSpace(string(out)))
}
```

**Step 2: Update `fetchPRInfo` to inject `GH_HOST`**

Find (around line 6585):
```go
cmd := exec.Command(ghPath, "pr", "view", "--json", "number,title,state,url,statusCheckRollup")
cmd.Dir = worktreePath
out, err := cmd.Output()
```

Replace with:
```go
cmd := exec.Command(ghPath, "pr", "view", "--json", "number,title,state,url,statusCheckRollup")
cmd.Dir = worktreePath
if host := ghHostFromDir(worktreePath); host != "" {
    cmd.Env = append(os.Environ(), "GH_HOST="+host)
}
out, err := cmd.Output()
```

**Step 3: Update `resolvePRBranch` to inject `GH_HOST`**

Find (around line 6880):
```go
cmd := exec.CommandContext(ctx, "gh", "pr", "view", prNum,
    "--json", "headRefName,title",
    "--jq", ".headRefName+\"\\t\"+.title")
cmd.Dir = repoDir
output, err := cmd.CombinedOutput()
```

Replace with:
```go
cmd := exec.CommandContext(ctx, "gh", "pr", "view", prNum,
    "--json", "headRefName,title",
    "--jq", ".headRefName+\"\\t\"+.title")
cmd.Dir = repoDir
if host := ghHostFromDir(repoDir); host != "" {
    cmd.Env = append(os.Environ(), "GH_HOST="+host)
}
output, err := cmd.CombinedOutput()
```

**Step 4: Build and test**

```bash
go build ./...
go test ./internal/ui/... 2>&1 | grep -v '^ok' | grep -v '^?'
```

Expected: only the two known pre-existing failures.

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): inject GH_HOST from git remote when calling gh CLI"
```

---

## Task 6: Update `update.go` repo constants

**Files:**
- Modify: `internal/update/update.go:22` (`GitHubRepo`)
- Modify: `internal/update/update.go:356–365` (`homebrewUpgradeHint` Cellar paths + command)

**Step 1: Update `GitHubRepo`**

Find:
```go
GitHubRepo = "mnicholson/hangar"
```

Replace with:
```go
GitHubRepo = "sjoeboo/hangar"
```

**Step 2: Update Homebrew hint command**

Find (around line 363):
```go
return "brew upgrade mnicholson/tap/hangar", true
```

Replace with:
```go
return "brew upgrade sjoeboo/tap/hangar", true
```

**Step 3: Build**

```bash
go build ./...
go test ./internal/update/... -v
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/update/update.go
git commit -m "fix: update repo constants to sjoeboo/hangar for open-source migration"
```

---

## Task 7: Update feedback URLs in runtime code

**Files:**
- Modify: `cmd/hangar/main.go:2454`

**Step 1: Update feedback URL**

Find:
```go
fmt.Println("Feedback: https://ghe.spotify.net/mnicholson/hangar/issues")
```

Replace with:
```go
fmt.Println("Feedback: https://github.com/sjoeboo/hangar/issues")
```

**Step 2: Build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/hangar/main.go
git commit -m "fix: update feedback URL to github.com/sjoeboo/hangar"
```

---

## Task 8: Update shell scripts and goreleaser

**Files:**
- Modify: `uninstall.sh`
- Modify: `install.sh`
- Modify: `.goreleaser.yml`

**Step 1: Update `uninstall.sh`**

Find (around line 306):
```bash
echo "Feedback: https://ghe.spotify.net/mnicholson/hangar/issues"
```

Replace with:
```bash
echo "Feedback: https://github.com/sjoeboo/hangar/issues"
```

Also update the header comments at the top of the file — find all `ghe.spotify.net/mnicholson/hangar` and replace with `github.com/sjoeboo/hangar`.

**Step 2: Update `install.sh`**

Replace all occurrences of:
- `ghe.spotify.net/mnicholson/hangar` → `github.com/sjoeboo/hangar`
- `git.spotify.net/mnicholson/hangar` → `github.com/sjoeboo/hangar`
- `ghe.spotify.net/mnicholson/hangar` in the `go.mod` check → `github.com/sjoeboo/hangar`

**Step 3: Update `.goreleaser.yml`**

Replace all occurrences:
- `ghe.spotify.net/mnicholson/hangar` → `github.com/sjoeboo/hangar`
- `mnicholson/tap/hangar` → `sjoeboo/tap/hangar` (Homebrew tap line)
- `mnicholson/hangar` → `sjoeboo/hangar` (standalone, if any)

**Step 4: Verify no `ghe.spotify.net` remains in scripts/config**

```bash
grep -r 'ghe\.spotify\.net\|mnicholson/hangar' uninstall.sh install.sh .goreleaser.yml
```

Expected: no matches.

**Step 5: Commit**

```bash
git add uninstall.sh install.sh .goreleaser.yml
git commit -m "fix: update all ghe.spotify.net references in scripts and goreleaser config"
```

---

## Final Verification

```bash
# No ghe.spotify.net left in runtime Go code
grep -r 'ghe\.spotify\.net' --include='*.go' . | grep -v 'go.mod\|go.sum\|docs/plans'

# No mnicholson/hangar left in runtime code (update.go, main.go)
grep -r 'mnicholson/hangar' --include='*.go' . | grep -v 'go.mod\|go.sum\|docs/plans\|vendor'

# Full test suite
go test ./...
```

Expected for grep: no matches outside of `go.mod` / docs (module path migration is out of scope).
Expected for tests: pass with only the two known pre-existing failures.
