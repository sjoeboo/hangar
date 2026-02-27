# Worktree Default + Ghost Item Progress

**Date:** 2026-02-27

## Problem

1. Worktree creation is not the default when starting a new session, even though it is the preferred workflow.
2. When a worktree is created in the background (can take 30–60s on large monorepos), there is no persistent progress indicator — only a transient error-bar message that auto-dismisses.

## Solution

### Change 1 — Make worktree the default

In `internal/ui/newdialog.go`, set `worktreeEnabled: true` as the default in both:
- `NewDialog()` initializer
- `Reset()` / `ShowInGroup()` reset path

### Change 2 — Ghost item in session list

Add a `pendingWorktrees []pendingWorktreeItem` slice to the `Home` struct. Each entry represents an in-progress async worktree creation.

```go
type pendingWorktreeItem struct {
    branchName string
    groupPath  string
    startedAt  time.Time
}
```

**Lifecycle:**
- When async worktree creation starts: append to `pendingWorktrees`, remove the `setError("creating worktree…")` call
- When `worktreeCreatedForNewSessionMsg` arrives (success or error): remove the matching entry

**Rendering:**
- Ghost items are NOT part of `flatItems`, so the cursor never lands on them (skip behavior is automatic)
- `renderSessionList` injects ghost rows at the correct group position after building rows from `flatItems`
- Ghost row uses dimmed style + Braille spinner from existing `animationFrame`, e.g.: `⠙ Creating worktree: my-feature-branch…`
- No new ticker needed — `animationFrame` already advances via `statusUpdateMsg`

**Fork path:**
- Same treatment for `worktreeCreatedForForkMsg`: add/remove from `pendingWorktrees` using source session's group
