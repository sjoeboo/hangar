# Implementation Plan: Hangar TUI Improvements

## What We're Building

A tab navigation bar that makes Sessions, PRs, and Todos discoverable as first-class views,
a redesigned status filter bar that replaces cryptic `!@#$` hints with labeled clickable pills,
and full mouse support across tabs, filter pills, and the session list (which already partially exists).

## Decisions Made

| # | Decision | Choice | Category |
|---|----------|--------|----------|
| 1 | Tab Navigation Model | Option B: Extensible Nav Bar (Sessions, PRs, Todos) | IA |
| 2 | Tab Visual Style | Option A: Rounded Pills (matches existing tmux tab style) | Visual |
| 3 | Filter Bar Redesign | Option A: Labeled Status Pills with inline key hints | Visual |
| 4 | Mouse Support Scope | Option C: Tabs + Filter Pills + Session Click (already exists) | Interaction |

## Implementation Steps

### Step 1 — Add `viewMode` values and tab model

- Add `"todos"` as a third `viewMode` value (currently only `""` and `"prs"`)
- Promote the `TodoDialog` overlay to a full-screen view rendered when `viewMode == "todos"`
  - The existing `t` key shortcut becomes the same as switching to the Todos tab
- Add `Tab` / `Shift+Tab` key bindings to cycle through views (Sessions → PRs → Todos → Sessions)
- Keep existing `P` key (switches to PRs) and `t` key (switches to Todos) as shortcuts

### Step 2 — Render the tab pill row

In `home.go` `View()`, render a new tab row **above** the filter row using the rounded pill style.

```
 Sessions  ·  PRs  ·  Todos
```

- Active tab: cyan background (`ColorCyan` / `#0891b2`), dark text
- Inactive tabs: dimmed background (`#252d40`), muted text
- Separator dots between pills (matching existing tmux window tab style)
- Only show the tab row when NOT inside a dialog/overlay

Track the rendered column positions of each tab for mouse hit-testing (Step 4).

### Step 3 — Redesign the filter bar

Replace the current filter row:
```
All  ● 1  ◐ 2  ○ 0    !@#$ filter · 0 all
```

With labeled pills:
```
 All   ● Running !   ◐ Waiting @   ○ Idle #
```

- `All` pill: active (highlighted) when no filter is set; click or `0` to clear
- `● Running !` pill: green background when active; `!` superscript shows keyboard shortcut
- `◐ Waiting @` pill: yellow background when active; `@` superscript
- `○ Idle #` pill: gray background when active; `#` superscript
- Error pill (`✕ Errors $`): only shown when `errored > 0` (same as current behavior)
- Remove the `"  !@#$ filter · 0 all"` hint text entirely

Track the rendered column positions of each pill for mouse hit-testing (Step 4).

### Step 4 — Wire up mouse support

Handle `tea.MouseMsg` in `Update()`. The session list already responds to mouse clicks — extend to cover the new elements.

**Tab row hit-testing** (row 0 of content area, after title bar):
- On `tea.MouseLeft` click: compare `msg.X` against stored tab start/end columns
- Switch `viewMode` to the clicked tab's view

**Filter pill hit-testing** (row 1 of content area):
- On `tea.MouseLeft` click: compare `msg.X` against stored pill start/end columns
- Set `statusFilter` to the clicked pill's status (or clear if "All")
- Rebuild flat items

**Session list** (already has mouse support — verify it still works after layout changes):
- Single click → move cursor to that row
- Double click → attach (Enter equivalent)

### Step 5 — Update help overlay and key bindings table

In `internal/ui/help.go`, add:
- `Tab` / `Shift+Tab` → Switch view
- `1` / `2` / `3` as optional shortcuts for Sessions / PRs / Todos (if not conflicting)
- Remove or update the `!@#$ filter` hint (it's now visible in the UI itself)

### Step 6 — Handle Todos tab content

When `viewMode == "todos"`, render the todo board full-screen instead of as an overlay:
- Reuse existing `TodoDialog` rendering logic
- Hide the session list + preview pane
- `Escape` or `t` returns to Sessions view (same as existing dialog dismiss)

### Step 7 — Tests and cleanup

- Verify existing `TestNewDialog_*` tests still pass (known pre-existing failures are OK)
- Add tests for tab switching via keyboard
- Add tests for mouse click on tab positions
- Verify PR overview still works (existing `viewMode == "prs"` path)

## Files to Change

| File | What changes |
|------|-------------|
| `internal/ui/home.go` | Tab row render, filter pill render, mouse hit-testing, `viewMode` values, key bindings |
| `internal/ui/styles.go` | New `lipgloss.Style` vars for tab pills (pre-compile at module level) |
| `internal/ui/help.go` | Update key bindings table |
| `internal/ui/todo_dialog.go` | Full-screen render mode when called as a view (not overlay) |

## Decision History

All decision documents: `.decisions/index.html`
