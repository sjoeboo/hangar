# Bulk Session Management Design

**Date:** 2026-02-26
**Branch:** bulk-session-mgmt
**Status:** Approved

## Overview

Add a vim-style visual mode to Hangar that lets users select multiple sessions and apply bulk actions: delete (with worktree cleanup), send message, or restart.

## Entry & Exit

- `V` toggles bulk-select mode on/off
- `Esc` exits bulk-select mode
- Entering mode clears any prior selection
- Exiting mode clears selection

## State

Two new fields on the `Home` struct:

```go
bulkSelectMode     bool
selectedSessionIDs map[string]bool
```

Groups are not selectable. When the cursor is on a group row, `space` is a no-op. Navigation (`j/k`) behaves identically in both modes.

## Visual Rendering

In bulk-select mode, each session row gains a checkbox prefix:
- Unselected: `[ ]`
- Selected: `[x]`

The existing cursor highlight is additive — the checkbox appears alongside it.

Status bar shows a mode banner when active:
```
VISUAL  3 selected  ·  space:toggle  d:delete  x:message  R:restart  Esc:cancel
```

## Actions

### Delete (`d`)
- Shows a new `ConfirmBulkDeleteSessions` dialog variant listing selected session names, noting which have worktrees (e.g., `my-feature [worktree]`)
- On confirmation: kills each session and runs worktree cleanup (reusing existing `deleteSession` logic)
- All deleted sessions pushed to undo stack as individual entries (Ctrl+Z restores one at a time)
- Exits bulk mode after completion

### Send Message (`x`)
- Opens the existing compose dialog unchanged
- On send: dispatches the message to all selected sessions
- Exits bulk mode after sending

### Restart (`R`)
- Simple confirm: *"Restart N sessions?"* (no per-session list needed)
- Restarts fire in parallel
- Exits bulk mode after completion

### Fallthrough behavior
If zero sessions are selected when an action key is pressed, it falls through to normal single-session behavior. This preserves backwards compatibility — `d` on a focused session still works if nothing is checked.

## Wiring in `home.go`

| Touchpoint | Change |
|---|---|
| Key routing | `V` added to main handler; `space` with mode-check guard; `d`/`x`/`R` check `bulkSelectMode` |
| Mouse guard | Bulk mode added to "block mouse when dialog visible" check |
| View rendering | Session row prefix shows `[ ]`/`[x]` when `bulkSelectMode` is true; status bar shows mode banner |
| `confirmDialog.go` | New `ConfirmBulkDeleteSessions` type with multi-line session list body |
| Result messages | New `bulkDeleteResultMsg`, `bulkRestartResultMsg`; send reuses existing result type |

No new dialog structs. Extends existing `ConfirmDialog` and reuses the send-text compose flow.

## Non-Goals

- Selecting group headers to select all sessions within (not in scope)
- Bulk move to group (not in scope for this iteration)
- Batch undo (individual undo entries per deleted session, same as today)
