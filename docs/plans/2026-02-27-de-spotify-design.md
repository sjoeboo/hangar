# De-Spotifying Hangar — Design

**Date:** 2026-02-27
**Status:** Approved

## Context

Hangar is being open-sourced at `github.com/sjoeboo/hangar`. Three runtime
issues need to be resolved before the repository can be made public. Go module
path migration is out of scope for this change.

---

## Issue 1: Configurable Remote Labels

### Problem

`normalizeRemoteURL` (`internal/ui/home.go:6625`) hardcodes `ghe.spotify.net`
as a known host with a short `"ghe:"` label. Any other GHE or self-hosted
instance would fall through to the raw URL, which is ugly in the preview pane.

### Design

Add a `RemoteLabels map[string]string` field to `UserConfig`:

```toml
# ~/.hangar/config.toml
[remote_labels]
"ghe.spotify.net" = "github"
```

`github.com → "github"` stays hardcoded as the built-in default.
Custom entries override the fallback for any other host.

**Signature change:**

```go
// Before
func normalizeRemoteURL(u string) string

// After
func normalizeRemoteURL(u string, extraLabels map[string]string) string
```

**Logic order:**
1. Hardcoded `github.com` → `"github"` (SSH + HTTPS)
2. Walk `extraLabels` — check if host of `u` matches any key; if so use the
   configured label
3. Return raw URL unchanged

Callers pass `h.cfg.RemoteLabels`. The existing call in
`update_handlers.go:794` already has access to `h`.

Test updates: replace the hardcoded `ghe.spotify.net` cases in
`styles_test.go` with table-driven cases that pass an explicit label map.

---

## Issue 2: `gh` CLI Host Inference

### Problem

`fetchPRInfo` and `resolvePRBranch` run `gh pr view` without setting `GH_HOST`.
When `gh` is authenticated to a GHE instance as its default host, running it
in a `github.com` repo silently fails or uses the wrong host. Conversely, a
`github.com`-primary `gh` install fails on GHE repos.

`gh` respects `GH_HOST` env var to override the target host.

### Design

Add a helper that extracts the remote hostname from a repo directory:

```go
// ghHostFromDir returns the hostname of the origin remote for dir,
// or "" if it cannot be determined.
func ghHostFromDir(dir string) string {
    out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
    if err != nil {
        return ""
    }
    return extractRemoteHost(strings.TrimSpace(string(out)))
}

// extractRemoteHost parses a git remote URL and returns just the hostname.
//   git@github.com:user/repo.git  → "github.com"
//   https://github.com/user/repo  → "github.com"
func extractRemoteHost(u string) string {
    // SSH: git@HOST:...
    if after, ok := strings.CutPrefix(u, "git@"); ok {
        host, _, _ := strings.Cut(after, ":")
        return host
    }
    // HTTPS
    if after, ok := strings.CutPrefix(u, "https://"); ok {
        host, _, _ := strings.Cut(after, "/")
        return host
    }
    if after, ok := strings.CutPrefix(u, "http://"); ok {
        host, _, _ := strings.Cut(after, "/")
        return host
    }
    return ""
}
```

In `fetchPRInfo`, derive the host and inject into the command env:

```go
func fetchPRInfo(sessionID, worktreePath, ghPath string) prFetchedMsg {
    cmd := exec.Command(ghPath, "pr", "view", ...)
    cmd.Dir = worktreePath
    if host := ghHostFromDir(worktreePath); host != "" {
        cmd.Env = append(os.Environ(), "GH_HOST="+host)
    }
    ...
}
```

Same pattern in `resolvePRBranch` (which has `repoDir`).

The extra `git remote get-url origin` subprocess is acceptable — both
functions are already doing network I/O and are not on the render hot path.

---

## Issue 3: Repository Constant and URL Updates

### Runtime code

| File | Old | New |
|------|-----|-----|
| `internal/update/update.go:22` | `"mnicholson/hangar"` | `"sjoeboo/hangar"` |
| `internal/update/update.go:363` | `brew upgrade mnicholson/tap/hangar` | `brew upgrade sjoeboo/tap/hangar` |
| `cmd/hangar/main.go:2454` | `https://ghe.spotify.net/mnicholson/hangar/issues` | `https://github.com/sjoeboo/hangar/issues` |

### Shell scripts (non-runtime, updated in same PR)

| File | Change |
|------|--------|
| `uninstall.sh:306` | feedback URL → `https://github.com/sjoeboo/hangar/issues` |
| `install.sh` | all `ghe.spotify.net` refs → `github.com/sjoeboo` |
| `.goreleaser.yml` | all `ghe.spotify.net` + `mnicholson` refs updated |

### Docs (out of scope for this PR, separate pass)

`README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `docs/`, `skills/hangar/` —
bulk find-and-replace after the repo migrates.

---

## Files Changed (runtime)

- `internal/session/userconfig.go` — add `RemoteLabels map[string]string`
- `internal/ui/home.go` — update `normalizeRemoteURL` signature + add
  `ghHostFromDir` + `extractRemoteHost` helpers; update `resolvePRBranch`
- `internal/ui/update_handlers.go` — pass `h.cfg.RemoteLabels` to
  `normalizeRemoteURL`
- `internal/ui/styles_test.go` — update `TestNormalizeRemoteURL`
- `internal/update/update.go` — update `GitHubRepo` constant + Homebrew hint
- `cmd/hangar/main.go` — update feedback URL
- `uninstall.sh`, `install.sh`, `.goreleaser.yml` — URL updates
