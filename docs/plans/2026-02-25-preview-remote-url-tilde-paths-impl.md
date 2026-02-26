# Preview Pane: Remote URL + Tilde Paths — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve the worktree session preview pane: add a normalized `Remote:` line, drop the redundant `Repo:` line, move PR+Checks to the top, and tilde-compress all displayed paths.

**Architecture:** All changes are in `internal/ui/home.go`. Remote URL is lazily fetched via a `git config` subprocess (same debounce/cache pattern as dirty status). Path shortening is a new helper wrapping the existing `truncatePath`.

**Tech Stack:** Go, Bubble Tea, lipgloss, `os/exec` for git subprocess.

---

### Task 1: Add `shortenPath` helper

**Files:**
- Modify: `internal/ui/home.go` (near `truncatePath` at line 8276)

**Step 1: Add the function immediately after `truncatePath`**

```go
// shortenPath replaces the home directory prefix with ~ then truncates to fit maxLen.
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

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```
Expected: no errors.

**Step 3: Write unit tests**

Add to `internal/ui/styles_test.go` (or `home_test.go` if that file tests helpers):

```go
func TestShortenPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input    string
		maxLen   int
		wantPrefix string
	}{
		{home + "/code/hangar", 100, "~/"},
		{home, 100, "~"},
		{"/tmp/other", 100, "/tmp/"},
		{home + "/very/long/path/that/exceeds/the/limit/yes/it/does", 20, "~/"},
	}
	for _, tt := range tests {
		got := shortenPath(tt.input, tt.maxLen)
		if !strings.HasPrefix(got, tt.wantPrefix) {
			t.Errorf("shortenPath(%q, %d) = %q, want prefix %q", tt.input, tt.maxLen, got, tt.wantPrefix)
		}
	}
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/... -run TestShortenPath -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/styles_test.go
git commit -m "feat: add shortenPath helper to tilde-compress paths in preview"
```

---

### Task 2: Apply `shortenPath` to existing path displays

**Files:**
- Modify: `internal/ui/home.go` — two call sites

**Step 1: Replace `truncatePath` with `shortenPath` for the 📁 project path line**

At line ~7405, change:
```go
pathStr := truncatePath(selected.ProjectPath, width-4)
```
to:
```go
pathStr := shortenPath(selected.ProjectPath, width-4)
```

**Step 2: Replace `truncatePath` with `shortenPath` for the worktree `Path:` line**

At line ~7535, change:
```go
wtPath := truncatePath(selected.WorktreePath, width-4-9)
```
to:
```go
wtPath := shortenPath(selected.WorktreePath, width-4-9)
```

**Step 3: Build and verify**

```bash
go build ./...
```
Expected: no errors.

**Step 4: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat: tilde-compress project path and worktree path in preview pane"
```

---

### Task 3: Add `normalizeRemoteURL` helper

**Files:**
- Modify: `internal/ui/home.go` (near `truncatePath`/`shortenPath`)

**Step 1: Add the function**

```go
// normalizeRemoteURL converts a git remote URL to a short human-readable form.
//   git@github.com:user/repo.git       -> "github: user/repo"
//   git@ghe.spotify.net:user/repo.git  -> "ghe: user/repo"
//   https://github.com/user/repo.git   -> "github: user/repo"
//   https://ghe.spotify.net/user/repo  -> "ghe: user/repo"
//   anything else                      -> returned unchanged
func normalizeRemoteURL(u string) string {
	// SSH format: git@HOST:PATH
	if rest, ok := strings.CutPrefix(u, "git@github.com:"); ok {
		return "github: " + strings.TrimSuffix(rest, ".git")
	}
	if rest, ok := strings.CutPrefix(u, "git@ghe.spotify.net:"); ok {
		return "ghe: " + strings.TrimSuffix(rest, ".git")
	}
	// HTTPS format
	for _, prefix := range []string{"https://github.com/", "http://github.com/"} {
		if rest, ok := strings.CutPrefix(u, prefix); ok {
			return "github: " + strings.TrimSuffix(rest, ".git")
		}
	}
	for _, prefix := range []string{"https://ghe.spotify.net/", "http://ghe.spotify.net/"} {
		if rest, ok := strings.CutPrefix(u, prefix); ok {
			return "ghe: " + strings.TrimSuffix(rest, ".git")
		}
	}
	return u
}
```

**Step 2: Write unit tests**

Add to `internal/ui/styles_test.go`:

```go
func TestNormalizeRemoteURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:mnicholson/hangar.git", "github: mnicholson/hangar"},
		{"git@ghe.spotify.net:squad/service.git", "ghe: squad/service"},
		{"https://github.com/mnicholson/hangar.git", "github: mnicholson/hangar"},
		{"https://github.com/mnicholson/hangar", "github: mnicholson/hangar"},
		{"https://ghe.spotify.net/squad/service.git", "ghe: squad/service"},
		{"https://ghe.spotify.net/squad/service", "ghe: squad/service"},
		{"git@gitlab.com:user/repo.git", "git@gitlab.com:user/repo.git"}, // passthrough
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

**Step 3: Run tests**

```bash
go test ./internal/ui/... -run TestNormalizeRemoteURL -v
```
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/ui/home.go internal/ui/styles_test.go
git commit -m "feat: add normalizeRemoteURL helper for preview pane remote display"
```

---

### Task 4: Add remote URL cache to home model

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add cache fields to the model struct**

After line 198 (the `worktreeDirtyMu` field), add:

```go
// Worktree remote URL cache (lazy, 5m TTL)
worktreeRemoteCache   map[string]string    // sessionID -> normalized remote URL ("" = none)
worktreeDirtyRemoteCacheTs map[string]time.Time // sessionID -> cache timestamp (reuse naming pattern)
worktreeRemoteMu      sync.Mutex           // Protects remote cache maps
```

Wait — use consistent naming. Add:

```go
// Worktree remote URL cache (lazy, 5m TTL)
worktreeRemoteCache   map[string]string    // sessionID -> normalized remote URL
worktreeRemoteCacheTs map[string]time.Time // sessionID -> cache timestamp
worktreeRemoteMu      sync.Mutex
```

**Step 2: Initialize in `newHomeModel`**

After line 544 (`worktreeDirtyCacheTs: make(...)`), add:

```go
worktreeRemoteCache:   make(map[string]string),
worktreeRemoteCacheTs: make(map[string]time.Time),
```

**Step 3: Add the result message type**

Near the `worktreeDirtyCheckMsg` type definition (around line 430), add:

```go
// worktreeRemoteCheckMsg is sent when an async remote URL fetch completes
type worktreeRemoteCheckMsg struct {
	sessionID string
	remoteURL string // empty string if none found
	err       error
}
```

**Step 4: Build and verify**

```bash
go build ./internal/ui/...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat: add worktree remote URL cache fields to home model"
```

---

### Task 5: Fetch remote URL in the debounce handler

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add fetch trigger in `previewDebounceMsg` handler**

After the PR status check block (after line ~2792), add a new block inside the `if inst != nil` block:

```go
// Remote URL fetch (lazy, 5m TTL)
if inst.IsWorktree() && inst.WorktreePath != "" {
    h.worktreeRemoteMu.Lock()
    cacheTs, hasCached := h.worktreeRemoteCacheTs[inst.ID]
    needsRemote := !hasCached || time.Since(cacheTs) > 5*time.Minute
    if needsRemote {
        h.worktreeRemoteCacheTs[inst.ID] = time.Now() // Prevent duplicate fetches
    }
    h.worktreeRemoteMu.Unlock()
    if needsRemote {
        sid := inst.ID
        wtPath := inst.WorktreePath
        cmds = append(cmds, func() tea.Msg {
            out, err := exec.Command("git", "-C", wtPath, "config", "--get", "remote.origin.url").Output()
            url := strings.TrimSpace(string(out))
            return worktreeRemoteCheckMsg{sessionID: sid, remoteURL: normalizeRemoteURL(url), err: err}
        })
    }
}
```

**Step 2: Handle `worktreeRemoteCheckMsg` in `Update`**

Near the `worktreeDirtyCheckMsg` case handler (around line 2812), add:

```go
case worktreeRemoteCheckMsg:
    h.worktreeRemoteMu.Lock()
    h.worktreeRemoteCache[msg.sessionID] = msg.remoteURL
    h.worktreeRemoteCacheTs[msg.sessionID] = time.Now()
    h.worktreeRemoteMu.Unlock()
    return h, nil
```

**Step 3: Build and verify**

```bash
go build ./...
```
Expected: no errors. Check that `os/exec` is already imported (it should be); if not, add `"os/exec"` to the imports.

**Step 4: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat: lazily fetch and cache worktree remote URL in preview debounce handler"
```

---

### Task 6: Reorder and update the worktree preview section

**Files:**
- Modify: `internal/ui/home.go` — the `renderPreviewPane` worktree section (lines ~7434–7566)

This is the core UI change. The current order is:
`Branch → PR → Checks → Repo → Path → Status → Finish`

New order:
`PR → Checks → Branch → Remote → Path → Status → Finish`

**Step 1: Restructure the worktree section**

Replace the entire worktree section (lines 7434–7566) with:

```go
// Worktree info section (for sessions running in git worktrees)
if selected.IsWorktree() {
    wtHeader := renderSectionDivider("Worktree", width-4)
    b.WriteString(wtHeader)
    b.WriteString("\n")

    wtLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
    wtBranchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
    wtValueStyle := lipgloss.NewStyle().Foreground(ColorText)
    wtHintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
    wtKeyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

    // PR status (from gh CLI, lazy-cached with 60s TTL)
    if h.ghPath != "" {
        h.prCacheMu.Lock()
        pr, hasPR := h.prCache[selected.ID]
        _, hasTs := h.prCacheTs[selected.ID]
        h.prCacheMu.Unlock()

        if !hasTs || (hasTs && !hasPR) {
            b.WriteString(wtLabelStyle.Render("PR:      "))
            b.WriteString(lipgloss.NewStyle().Foreground(ColorComment).Render("checking..."))
            b.WriteString("\n")
        } else if pr != nil {
            stateStyle := lipgloss.NewStyle()
            stateLabel := strings.ToLower(pr.State)
            switch pr.State {
            case "OPEN":
                stateStyle = stateStyle.Foreground(ColorGreen)
            case "DRAFT":
                stateStyle = stateStyle.Foreground(ColorComment)
            case "MERGED":
                stateStyle = stateStyle.Foreground(ColorPurple)
            case "CLOSED":
                stateStyle = stateStyle.Foreground(ColorRed)
            }
            prNumStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Underline(true)
            titleMax := width - 4 - 9 - 6 - len(stateLabel) - 3
            title := pr.Title
            if titleMax > 10 && runewidth.StringWidth(title) > titleMax {
                title = runewidth.Truncate(title, titleMax, "...")
            }
            b.WriteString(wtLabelStyle.Render("PR:      "))
            b.WriteString(prNumStyle.Render(fmt.Sprintf("#%d", pr.Number)))
            b.WriteString(" ")
            b.WriteString(stateStyle.Render(stateLabel))
            b.WriteString(wtValueStyle.Render(" · " + title))
            b.WriteString("\n")
            if pr.URL != "" {
                urlStyle := lipgloss.NewStyle().Foreground(ColorComment)
                urlMax := width - 4 - 9
                displayURL := pr.URL
                if runewidth.StringWidth(displayURL) > urlMax && urlMax > 15 {
                    displayURL = runewidth.Truncate(displayURL, urlMax, "…")
                }
                b.WriteString(wtLabelStyle.Render("         "))
                b.WriteString(urlStyle.Render(displayURL))
                b.WriteString("\n")
            }
            if pr.HasChecks {
                b.WriteString(wtLabelStyle.Render("Checks:  "))
                var parts []string
                if pr.ChecksFailed > 0 {
                    parts = append(parts, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✗ %d failed", pr.ChecksFailed)))
                }
                if pr.ChecksPending > 0 {
                    parts = append(parts, lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("● %d running", pr.ChecksPending)))
                }
                if pr.ChecksPassed > 0 {
                    parts = append(parts, lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("✓ %d passed", pr.ChecksPassed)))
                }
                b.WriteString(strings.Join(parts, "  "))
                b.WriteString("\n")
            }
        }
        // pr == nil means no PR found; omit line silently
    }

    // Branch
    if selected.WorktreeBranch != "" {
        b.WriteString(wtLabelStyle.Render("Branch:  "))
        b.WriteString(wtBranchStyle.Render(selected.WorktreeBranch))
        b.WriteString("\n")
    }

    // Remote URL (lazy-cached, 5m TTL)
    h.worktreeRemoteMu.Lock()
    remoteURL, hasRemote := h.worktreeRemoteCache[selected.ID]
    _, hasRemoteTs := h.worktreeRemoteCacheTs[selected.ID]
    h.worktreeRemoteMu.Unlock()
    if !hasRemoteTs {
        b.WriteString(wtLabelStyle.Render("Remote:  "))
        b.WriteString(lipgloss.NewStyle().Foreground(ColorComment).Render("checking..."))
        b.WriteString("\n")
    } else if hasRemote && remoteURL != "" {
        displayRemote := truncatePath(remoteURL, width-4-9)
        b.WriteString(wtLabelStyle.Render("Remote:  "))
        b.WriteString(wtValueStyle.Render(displayRemote))
        b.WriteString("\n")
    }

    // Worktree path (tilde-compressed)
    if selected.WorktreePath != "" {
        wtPath := shortenPath(selected.WorktreePath, width-4-9)
        b.WriteString(wtLabelStyle.Render("Path:    "))
        b.WriteString(wtValueStyle.Render(wtPath))
        b.WriteString("\n")
    }

    // Dirty status (lazy-cached, fetched via previewDebounce handler with 10s TTL)
    h.worktreeDirtyMu.Lock()
    isDirty, hasCached := h.worktreeDirtyCache[selected.ID]
    h.worktreeDirtyMu.Unlock()

    dirtyLabel := "checking..."
    dirtyStyle := wtValueStyle
    if hasCached {
        if isDirty {
            dirtyLabel = "dirty (uncommitted changes)"
            dirtyStyle = lipgloss.NewStyle().Foreground(ColorYellow)
        } else {
            dirtyLabel = "clean"
            dirtyStyle = lipgloss.NewStyle().Foreground(ColorGreen)
        }
    }
    b.WriteString(wtLabelStyle.Render("Status:  "))
    b.WriteString(dirtyStyle.Render(dirtyLabel))
    b.WriteString("\n")

    // Finish hint
    b.WriteString(wtHintStyle.Render("Finish:  "))
    b.WriteString(wtKeyStyle.Render("W"))
    b.WriteString(wtHintStyle.Render(" finish + cleanup"))
    b.WriteString("\n")
}
```

**Step 2: Build and verify**

```bash
go build ./...
```
Expected: no errors.

**Step 3: Run full test suite**

```bash
go test ./... 2>&1 | grep -v "^ok"
```
Expected: only the two known pre-existing failures (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`). No new failures.

**Step 4: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat: reorder worktree preview (PR first), add Remote line, drop Repo line"
```

---

### Task 7: Final verification

**Step 1: Run all tests**

```bash
go test ./...
```
Expected: same pre-existing failures only.

**Step 2: Build binary**

```bash
go build ./cmd/hangar
```
Expected: clean build.

**Step 3: Manual smoke test** (if hangar is running)

Navigate to a worktree session in the TUI. Verify:
- PR + Checks appear at the top of the Worktree section
- Branch appears below PR
- `Remote:` shows normalized form (e.g. `github: user/repo`)
- `Path:` shows tilde-compressed path (e.g. `~/.worktrees/...`)
- No `Repo:` line
- `📁` project path at the top of the preview is also tilde-compressed

**Step 4: Invoke finishing-a-development-branch skill**

```
superpowers:finishing-a-development-branch
```
