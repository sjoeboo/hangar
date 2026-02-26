# PR Status in Worktree Cleanup Dialog — Design

**Date:** 2026-02-25
**Branch:** cleanup-pr-status

## Problem

Two gaps in the worktree session lifecycle UX:

1. The `WorktreeFinishDialog` (triggered by `w`) shows branch and dirty status but not PR info — users can't see whether their PR is open/merged or if CI is passing before they clean up.
2. Pressing `d` on a worktree session shows a minimal `ConfirmDialog` that silently removes the worktree — no branch options, no PR context, and no dirty-state warning.

## Goals

- Show PR number, state, title, and CI check summary inside `WorktreeFinishDialog`.
- Route `d` on worktree sessions to `WorktreeFinishDialog` instead of `ConfirmDialog`, so all worktree lifecycle flows through one consistent dialog.

## Non-Goals

- No changes to non-worktree session deletion (`ConfirmDialog` unchanged).
- No new dialogs.
- No changes to PR fetching/caching logic.

## Design

### 1. `WorktreeFinishDialog` — PR Section

Add fields:
```go
prEntry  *prCacheEntry // nil = no PR found
prLoaded bool         // false = still fetching
```

Add method:
```go
func (d *WorktreeFinishDialog) SetPR(pr *prCacheEntry, loaded bool)
```

Render a PR block in `viewOptions()` below the branch/dirty section:

| State | Display |
|-------|---------|
| `prLoaded == false` | `PR: checking...` |
| `prLoaded == true, prEntry == nil` | `PR: none` |
| `prLoaded == true, prEntry != nil` | `PR #42 · OPEN · "Add user auth endpoint"` + CI line |

CI line format: `✓ 12 passed  ✗ 1 failed  ⟳ 2 pending` (omit zero counts).
State colored via existing `PRBadgeOpen/Merged/Closed` lipgloss styles.

### 2. `d` Key Routing

In `home.go`'s `d` handler:

```go
if item.Session.IsWorktree() {
    h.showWorktreeFinishDialog(item.Session)  // same path as 'w'
} else {
    h.confirmDialog.ShowDeleteSession(...)    // unchanged
}
```

### 3. PR Data Wiring

- When `WorktreeFinishDialog.Show()` is called, pass current cached PR data immediately (may be unloaded).
- When `prFetchedMsg` arrives in `Update()` and the dialog is visible with a matching session ID, call `SetPR()` to push fresh data — dialog re-renders automatically.

## Behaviour Summary

| Trigger | Session type | Result |
|---------|-------------|--------|
| `w` | worktree | `WorktreeFinishDialog` with PR info (unchanged entry point, now richer) |
| `d` | worktree | `WorktreeFinishDialog` with PR info (new routing) |
| `d` | non-worktree | `ConfirmDialog` (unchanged) |

## What Stays the Same

- Two-step dialog flow (options → confirm)
- Keep-branch toggle
- Async dirty check
- All PR fetching/caching (60s TTL, `gh` CLI)
- Non-worktree deletion
