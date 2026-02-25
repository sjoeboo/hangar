# PR Badge in Session Sidebar

**Date**: 2026-02-25
**Status**: Approved

## Summary

Add a `[#123]` badge to session rows in the sidebar for sessions that have an associated pull request in OPEN, MERGED, or CLOSED state.

## Rendered Output

```
├─ ● my-feature  claude  [pr-icon]  [#42]
├─ ○ old-work    claude  [merged-branch]  [#38]
```

## Specification

### When to show
- Session is a worktree session (`IsWorktree() == true`)
- PR cache has been populated for the session
- PR state is `OPEN`, `MERGED`, or `CLOSED` (not `DRAFT`, not nil)

### Colors
| State    | Color       |
|----------|-------------|
| `OPEN`   | `ColorGreen`  |
| `MERGED` | `ColorPurple` |
| `CLOSED` | `ColorRed`    |

### Position
Appended after the existing worktree branch badge at the end of the session row.

### Edge cases
- Cache not yet populated: no badge shown; appears on next render after async fetch completes
- `gh` not installed or not a worktree session: cache never populated, no badge shown
- `DRAFT` state: no badge shown

## Scope

### Not in scope
- Notification area (tmux status bar) — sidebar only
- Any new data fetching — uses existing `prCache` infrastructure

## Code Changes

1. **`internal/ui/styles.go`** — add `PRBadgeOpen`, `PRBadgeMerged`, `PRBadgeClosed` lipgloss styles
2. **`internal/ui/home.go`** — in `renderSessionItem()`, after the worktree badge block, build a `prBadge` string from `prCache` and append to the row format string
