# PR Overview View — Design

**Date**: 2026-02-26
**Status**: Approved

## Summary

Add a full-screen PR overview mode to Hangar, toggled with `P`, that shows only sessions with active PRs in a compact list with PR state and CI check status.

## Motivation

Users managing multiple worktrees need a focused view to quickly scan PR states and CI results across all active sessions without navigating the preview pane for each one individually.

## Approach

Option A: View mode flag on `Home`. Add a `viewMode` string field to `homeModel`. The existing `View()` function routes to `renderPROverview()` when in PR mode. Reuses the existing PR cache, fetch machinery, and style primitives — no new data layer needed.

This matches the existing kanban/todo full-screen toggle pattern.

## State Changes

Two new fields on `homeModel`:

```go
viewMode     string // "sessions" (default) or "prs"
prViewCursor int    // cursor position within the filtered PR list
```

## Rendering

`renderPROverview()` fills the full terminal with:

- Reused `renderHeader()` at top
- Column header line: `  PR    STATE    CHECKS    SESSION`
- One row per session that has a non-nil PR cache entry:
  ```
    #123  OPEN  ✓12 ✗1 ●2  my-feature-branch / Fix login timeout
  ```
  - `#123` — accent color, bold
  - `OPEN/DRAFT/MERGED/CLOSED` — state color (green/gray/purple/red)
  - `✓ ✗ ●` check counts in green/red/yellow (omitted if no checks)
  - Session title — bright if running, dim if idle
- Selected row uses existing highlight style
- Reused `renderHelpBar()` at bottom showing PR-view-specific hints

Sessions with no PR in cache are hidden entirely.

## Key Bindings (PR View)

| Key | Action |
|-----|---------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` | Attach to selected session |
| `o` | Open PR URL in browser |
| `r` | Force re-fetch PR data for all visible sessions |
| `P` or `esc` | Return to sessions view |

All other session-management keys (`n`, `d`, `w`, `x`, etc.) are ignored in this view.

## Data Flow

- On entering PR view: trigger fresh `fetchPRInfo` for any session with stale or missing cache entry
- `prFetchedMsg` updates are handled identically regardless of active view mode
- The filtered PR list is computed at render time from `prCache` — no separate data structure

## Access

- `P` (uppercase) from sessions view enters PR overview
- `P` or `esc` exits back to sessions view
- Lowercase `p` remains "new project" (unchanged)

## Files Changed

- `internal/ui/home.go` — add `viewMode`, `prViewCursor` fields; add `renderPROverview()`; wire key handler; add `P` hint to help bar
