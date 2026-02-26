# Kanban Todo View Design

**Date**: 2026-02-26
**Branch**: kanban-todo-view
**Status**: Approved

## Overview

Replace the existing linear list view in `TodoDialog` with a kanban board where columns represent todo status. All CRUD and session-linking behaviour is preserved; only the visual presentation and navigation model change.

## Approach

Extend the existing `TodoDialog` in place (Approach A). Remove `todoModeList` and add `todoModeKanban`. The three overlay modes (`todoModeNew`, `todoModeEdit`, `todoModeStatus`) are unchanged and continue to render on top of the kanban view.

## Data Model

### New cursor state in `TodoDialog`

```go
selectedCol  int   // index of focused column
selectedRow  []int // cursor row per column (len == number of visible columns)
```

Replaces the single `selectedIndex int` from the list view.

### Column structure (derived, not stored)

```go
type kanbanColumn struct {
    status TodoStatus
    label  string
    todos  []Todo
}
```

Built at render/key time from the full todos slice. No new storage or DB changes needed.

### Column ordering

```
[ Todo ]  [ In Progress ]  [ In Review ]  [ Done ]  [ Orphaned* ]
```

`Orphaned` column is only shown when orphaned todos exist. It is excluded from shift+arrow card movement.

## Layout & Rendering

Dialog width divided equally among visible columns (4 or 5), 1-character gap between them.

### Column header
```
In Progress   (3)
──────────────────
```
Label + count, underlined. Focused column header rendered in accent color.

### Card (one line per todo)
```
▌● Title text here                              ⬡
```
- `▌` — selection indicator (left border on selected card in focused column)
- Status icon + color (○ gray / ● cyan / ⟳ yellow / ✓ green / ! red)
- Title, truncated to column width
- `⬡` — session link indicator (right-aligned, only when `SessionID != ""`)

Non-focused column cards are rendered dimmed.

### Empty column
```
Todo
────
  (empty)
```

### Empty board (no todos at all)
```
No todos yet — press n to create one
```

## Key Bindings

### Navigation
| Key | Action |
|-----|--------|
| `←` / `h` | Move focus to left column |
| `→` / `l` | Move focus to right column |
| `↑` / `k` | Move cursor up within column |
| `↓` / `j` | Move cursor down within column |

Column focus does not wrap at edges. Cursor position per column is preserved when navigating away and back.

### Card actions
| Key | Action |
|-----|--------|
| `enter` | Open/attach session linked to todo |
| `e` | Edit card (`todoModeEdit` overlay) |
| `d` | Delete card |
| `s` | Status picker overlay (`todoModeStatus`) |
| `n` | New todo (`todoModeNew` overlay, pre-selects focused column's status) |

### Moving cards
| Key | Action |
|-----|--------|
| `shift+←` | Move card to previous status column |
| `shift+→` | Move card to next status column |

After a move, focus follows the card to its new column. No-op at boundaries (leftmost / rightmost non-orphaned column). Orphaned column excluded from shift+arrow.

### Dismiss
| Key | Action |
|-----|--------|
| `esc` / `t` | Close dialog |

## Edge Cases

- **Cursor persistence**: after save/refresh, restore cursor to same todo ID; clamp to last item if gone
- **New todo pre-selection**: `n` while focused on a column pre-sets the new todo's status to that column's status
- **Shift+arrow boundaries**: no-op at `todo` (leftmost) and `done` (rightmost movable) columns
- **Orphaned column disappears**: if user was focused on it, shift focus left to `done`
- **Single column populated**: other columns still render empty for layout stability

## Files to Change

| File | Change |
|------|--------|
| `internal/ui/todo_dialog.go` | Remove `todoModeList`, add `todoModeKanban`, new cursor state, kanban rendering, shift+arrow handling |
| `internal/ui/home.go` | Update any references to `todoModeList` |
