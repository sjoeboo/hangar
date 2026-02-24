# Mouse Support Design

**Date:** 2026-02-23
**Branch:** feature/mouse

## Overview

Add mouse support to the agent-deck TUI, enabling:
1. Click to navigate and attach sessions in the session list
2. Expand/collapse groups by clicking
3. Tmux mouse mode (window switching via status bar) enabled on attach via config toggle

`tea.WithMouseCellMotion()` is already wired in `main.go`, so mouse events are dispatched to `Update()` but currently silently dropped. This feature is purely additive: a new `case tea.MouseMsg:` handler.

## Session List Interactions

| Gesture | Target | Effect |
|---|---|---|
| Single left-click | Session row | Move cursor to that row |
| Double left-click | Session row | Attach to session (same as `Enter`) |
| Single left-click | Group header row | Toggle expand/collapse |
| Scroll wheel up/down | Anywhere in list | Scroll list |

Clicks in the right preview panel or outside list bounds are ignored.

## Coordinate Mapping

The list's first row index is computed at runtime:

```
listStartRow = 1 (header) + 1 (filter) + updateBannerHeight + maintenanceBannerHeight + 2 (panel title + underline)
```

Each item occupies exactly one terminal row. Given a click at `(x, y)`:

```
itemIndex = h.viewOffset + (y - listStartRow)
```

In dual-layout mode the left panel is `~35%` of terminal width; clicks with `x >= leftPanelWidth` are ignored. Stacked/single-layout uses full width.

## Double-Click Detection

Bubbletea has no native double-click. Two fields are added to the `Home` struct:

```go
lastClickTime  time.Time
lastClickIndex int
```

On each `MouseButtonLeft` press:
- Compute `itemIndex` from coordinates
- If `itemIndex == lastClickIndex` and `time.Since(lastClickTime) < 300ms` → double-click → attach
- Otherwise → single click → move cursor, update `lastClickTime` and `lastClickIndex`

The 300ms threshold matches standard OS double-click defaults.

## Tmux Mouse Mode Config Toggle

New config field:

```toml
# config.toml
mouse_mode = true  # enables tmux mouse support when attaching sessions (default: true)
```

When `mouse_mode = true`, `attachSession()` calls the existing `EnableMouseMode()` on the tmux session before handing the terminal over via `tea.Exec`. This runs `tmux set-option -t <session> mouse on`, enabling:
- Clicking tmux status bar to switch windows
- Scroll wheel inside panes

The tmux mouse state persists in the session and is re-applied on every attach. Default is `true`; keyboard-only users can set `mouse_mode = false`.

## Files to Modify

| File | Change |
|---|---|
| `internal/ui/home.go` | Add `lastClickTime`, `lastClickIndex` fields to `Home`; add `case tea.MouseMsg:` in `Update()`; add coordinate-mapping helper |
| `internal/config/config.go` | Add `MouseMode bool` field with `toml:"mouse_mode"` tag |
| `internal/session/attach.go` (or `home.go`) | Call `EnableMouseMode()` before `tea.Exec` when `config.MouseMode` is true |
| Config template / docs | Document `mouse_mode` option |
