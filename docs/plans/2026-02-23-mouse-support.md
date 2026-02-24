# Mouse Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add mouse support to the agent-deck TUI: single-click to move cursor, double-click to attach session, single-click to toggle group, scroll-wheel to scroll the list, and a `mouse_mode` config toggle that enables tmux mouse mode on attach.

**Architecture:** `tea.WithMouseCellMotion()` is already wired in `main.go`; mouse events are dispatched but silently dropped. We add a `case tea.MouseMsg:` handler in `Update()`, a `listItemAt(x, y)` coordinate-mapping helper, double-click detection via two new `Home` struct fields, and a `MouseMode *bool` config field in `TmuxSettings` that calls the existing `EnableMouseMode()` before `tea.Exec` on attach.

**Tech Stack:** Go, Bubbletea v1.3.10 (tea.MouseMsg API), TOML config, lipgloss layout constants.

---

### Task 1: Add `MouseMode` to `TmuxSettings`

**Files:**
- Modify: `internal/session/userconfig.go:712-722` (TmuxSettings struct)
- Modify: `internal/session/userconfig.go:724-727` (after GetInjectStatusLine accessor)

**Context:** `TmuxSettings` uses `*bool` with a named accessor for options that default to true. We follow the same pattern but default to `false` (opt-in). Config is read via `session.LoadUserConfig()` which returns a cached singleton.

**Step 1: Write the failing test**

In `internal/session/userconfig_test.go` (find the file first with `glob internal/session/*_test.go`), add:

```go
func TestTmuxSettingsGetMouseMode(t *testing.T) {
    // nil pointer → default false
    s := TmuxSettings{}
    if s.GetMouseMode() != false {
        t.Error("expected GetMouseMode() default to be false")
    }

    // explicit true
    tr := true
    s2 := TmuxSettings{MouseMode: &tr}
    if s2.GetMouseMode() != true {
        t.Error("expected GetMouseMode() to return true when set")
    }

    // explicit false
    fa := false
    s3 := TmuxSettings{MouseMode: &fa}
    if s3.GetMouseMode() != false {
        t.Error("expected GetMouseMode() to return false when set")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/session/... -run TestTmuxSettingsGetMouseMode -v
```
Expected: compile error (`GetMouseMode` undefined) or FAIL.

**Step 3: Add field and accessor**

In `userconfig.go`, add to `TmuxSettings` struct (after line 721, before closing `}`):

```go
// MouseMode controls whether tmux mouse mode is enabled when attaching sessions.
// When true, clicking the tmux status bar switches windows and scroll works in panes.
// Default: false (opt-in). Enable with mouse_mode = true in [tmux] config section.
MouseMode *bool `toml:"mouse_mode"`
```

After the `GetInjectStatusLine` function (around line 727), add:

```go
// GetMouseMode returns whether tmux mouse mode should be enabled on attach.
// Defaults to false if not set.
func (t TmuxSettings) GetMouseMode() bool {
    if t.MouseMode == nil {
        return false
    }
    return *t.MouseMode
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/session/... -run TestTmuxSettingsGetMouseMode -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/session/userconfig.go internal/session/userconfig_test.go
git commit -m "feat(config): add mouse_mode toggle to TmuxSettings"
```

---

### Task 2: Add click-tracking fields to `Home` struct

**Files:**
- Modify: `internal/ui/home.go:276-277` (after `lastNavigationTime`/`isNavigating`)

**Context:** No test needed — these are internal state fields used only by the handler added in Task 4. They need to exist before the handler is written.

**Step 1: Add fields after the navigation tracking block (around line 277)**

```go
// Mouse click tracking (double-click detection)
lastClickTime  time.Time
lastClickIndex int
```

The `time` package is already imported. `lastClickIndex` zero value (-1 would be better to distinguish "no prior click" but 0 is fine since we also check `lastClickTime.IsZero()`).

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): add mouse click-tracking fields to Home struct"
```

---

### Task 3: Write failing tests for `listItemAt()`

**Files:**
- Modify: `internal/ui/home_test.go`

**Context:** `listItemAt(x, y int) int` maps screen coordinates to a `flatItems` index. It returns `-1` for out-of-bounds clicks. Tests run in `package ui` with direct access to private fields. Tests call `NewHome()` and populate fields directly.

The list start row formula (screen coordinates, 0-indexed):
```
listStartRow = 1 (top bar) + 1 (filter bar) + updateBannerHeight + maintenanceBannerHeight + 2 (panel title+underline)
             = 4 + updateBannerHeight + maintenanceBannerHeight
```
Add 1 more if `h.viewOffset > 0` (the "more above" indicator occupies row 0 of the list area).

In dual layout (width ≥ 80), only clicks with `x < int(float64(h.width)*0.35)` hit the list.

**Step 1: Add tests to home_test.go**

```go
func TestListItemAt(t *testing.T) {
    makeItems := func(n int) []session.Item {
        items := make([]session.Item, n)
        for i := range items {
            items[i] = session.Item{Type: session.ItemTypeSession}
        }
        return items
    }

    t.Run("basic click hits correct item", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeItems(5)
        h.viewOffset = 0
        // listStartRow = 4 (no banners, viewOffset=0)
        // clicking row 4 → item 0, row 5 → item 1, etc.
        if got := h.listItemAt(5, 4); got != 0 {
            t.Errorf("row 4 → want 0, got %d", got)
        }
        if got := h.listItemAt(5, 6); got != 2 {
            t.Errorf("row 6 → want 2, got %d", got)
        }
    })

    t.Run("click above list returns -1", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeItems(5)
        if got := h.listItemAt(5, 0); got != -1 {
            t.Errorf("row 0 → want -1, got %d", got)
        }
        if got := h.listItemAt(5, 3); got != -1 {
            t.Errorf("row 3 → want -1, got %d", got)
        }
    })

    t.Run("click beyond items returns -1", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeItems(3)
        // listStartRow=4, items at rows 4,5,6 → row 7 is out of bounds
        if got := h.listItemAt(5, 7); got != -1 {
            t.Errorf("row beyond items → want -1, got %d", got)
        }
    })

    t.Run("dual layout: click in right panel returns -1", func(t *testing.T) {
        h := NewHome()
        h.width = 120 // dual layout (>=80)
        h.height = 30
        h.flatItems = makeItems(5)
        leftWidth := int(float64(120) * 0.35) // = 42
        // click at x=50 is in the right panel
        if got := h.listItemAt(50, 4); got != -1 {
            t.Errorf("right panel click → want -1, got %d", got)
        }
        // click at x=41 is in left panel
        if got := h.listItemAt(leftWidth-1, 4); got != 0 {
            t.Errorf("left panel click → want 0, got %d", got)
        }
    })

    t.Run("viewOffset shifts item mapping", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeItems(10)
        h.viewOffset = 3
        // viewOffset>0 adds 1 row for "more above" indicator
        // listStartRow = 4 + 1 = 5
        // row 5 → item 3 (viewOffset + 0)
        if got := h.listItemAt(5, 5); got != 3 {
            t.Errorf("row 5 with viewOffset=3 → want 3, got %d", got)
        }
    })
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/... -run TestListItemAt -v
```
Expected: compile error (`listItemAt` undefined).

---

### Task 4: Implement `listItemAt()`

**Files:**
- Modify: `internal/ui/home.go` (add new method, anywhere near the bottom of the file or near renderSessionList)

**Step 1: Add method**

```go
// listItemAt returns the flatItems index for the given screen coordinates (0-indexed),
// or -1 if the coordinates don't correspond to a visible list item.
func (h *Home) listItemAt(x, y int) int {
    // In dual layout, clicks in the right panel are ignored.
    if h.getLayoutMode() == LayoutModeDual {
        leftWidth := int(float64(h.width) * 0.35)
        if x >= leftWidth {
            return -1
        }
    }

    // Compute the screen row of the first list item.
    updateBannerHeight := 0
    if h.updateInfo != nil && h.updateInfo.Available {
        updateBannerHeight = 1
    }
    maintenanceBannerHeight := 0
    if h.maintenanceMsg != "" {
        maintenanceBannerHeight = 1
    }
    listStartRow := 1 + 1 + updateBannerHeight + maintenanceBannerHeight + 2
    if h.viewOffset > 0 {
        listStartRow++ // "more above" indicator occupies the first list row
    }

    if y < listStartRow {
        return -1
    }

    idx := h.viewOffset + (y - listStartRow)
    if idx < 0 || idx >= len(h.flatItems) {
        return -1
    }
    return idx
}
```

**Step 2: Run test to verify it passes**

```bash
go test ./internal/ui/... -run TestListItemAt -v
```
Expected: all subtests PASS.

**Step 3: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(ui): add listItemAt() coordinate-to-item mapper"
```

---

### Task 5: Write failing tests for the mouse handler

**Files:**
- Modify: `internal/ui/home_test.go`

**Context:** The handler will be a `handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd)` method called from `Update()`. Tests send `tea.MouseMsg` directly. In Bubbletea v1.x, the struct is `tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: ..., Y: ...}`.

**Step 1: Add tests**

```go
func TestHandleMouseMsg(t *testing.T) {
    makeSessionItems := func(n int) []session.Item {
        items := make([]session.Item, n)
        for i := range items {
            inst := &session.Instance{}
            items[i] = session.Item{Type: session.ItemTypeSession, Session: inst}
        }
        return items
    }

    t.Run("single left click moves cursor", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeSessionItems(5)
        h.cursor = 0

        msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 5}
        model, _ := h.Update(msg)
        updated := model.(*Home)
        // listStartRow=4, row 5 → item 1
        if updated.cursor != 1 {
            t.Errorf("expected cursor=1 after click row 5, got %d", updated.cursor)
        }
    })

    t.Run("click above list does not move cursor", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeSessionItems(5)
        h.cursor = 2

        msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 0}
        model, _ := h.Update(msg)
        updated := model.(*Home)
        if updated.cursor != 2 {
            t.Errorf("expected cursor unchanged at 2, got %d", updated.cursor)
        }
    })

    t.Run("double click on same item records tracking state", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeSessionItems(5)
        h.cursor = 0

        // First click
        msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 4}
        model, _ := h.Update(msg)
        h = model.(*Home)
        if h.lastClickIndex != 0 {
            t.Errorf("expected lastClickIndex=0 after first click, got %d", h.lastClickIndex)
        }
        if h.lastClickTime.IsZero() {
            t.Error("expected lastClickTime to be set after first click")
        }
    })

    t.Run("scroll wheel up decrements viewOffset", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeSessionItems(20)
        h.viewOffset = 5
        h.cursor = 5

        msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress, X: 5, Y: 10}
        model, _ := h.Update(msg)
        updated := model.(*Home)
        if updated.cursor >= 5 {
            t.Errorf("expected cursor to move up after wheel up, got %d", updated.cursor)
        }
    })

    t.Run("scroll wheel down increments cursor", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        h.flatItems = makeSessionItems(20)
        h.cursor = 0

        msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress, X: 5, Y: 10}
        model, _ := h.Update(msg)
        updated := model.(*Home)
        if updated.cursor <= 0 {
            t.Errorf("expected cursor to move down after wheel down, got %d", updated.cursor)
        }
    })

    t.Run("click on group item toggles group", func(t *testing.T) {
        h := NewHome()
        h.width = 120
        h.height = 30
        grp := &session.Group{Name: "test", Path: "test", Expanded: true}
        h.groupTree = session.NewGroupTree()
        h.groupTree.AddGroup(grp)
        h.flatItems = []session.Item{
            {Type: session.ItemTypeGroup, Group: grp, Path: "test"},
        }
        h.cursor = 0

        // Click on the group row
        msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 5, Y: 4}
        model, _ := h.Update(msg)
        updated := model.(*Home)
        // Group should now be collapsed
        if updated.groupTree.IsExpanded("test") {
            t.Error("expected group to be collapsed after click")
        }
    })
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/... -run TestHandleMouseMsg -v
```
Expected: tests fail (no mouse handler yet, clicks have no effect).

---

### Task 6: Implement `handleMouseMsg()` and wire into `Update()`

**Files:**
- Modify: `internal/ui/home.go`

**Context:** Check `session.NewGroupTree()` and `h.groupTree.IsExpanded()` existence — the test references them. If `NewGroupTree()` or `IsExpanded()` don't exist, check `internal/session/groups.go` for the actual constructor and expanded-state accessor and adjust test accordingly before implementing.

**Step 1: Add `handleMouseMsg` method**

Add this method to `home.go` (place it near `handleMainKey` for discoverability):

```go
// handleMouseMsg processes mouse events for the session list.
// It implements: single-click to move cursor, double-click to attach,
// group-click to toggle, and scroll-wheel to scroll.
func (h *Home) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
    // Only handle press events (ignore release/motion)
    if msg.Action != tea.MouseActionPress {
        return h, nil
    }

    switch msg.Button {
    case tea.MouseButtonWheelUp:
        if h.cursor > 0 {
            h.cursor--
            h.syncViewport()
        }
        return h, nil

    case tea.MouseButtonWheelDown:
        if h.cursor < len(h.flatItems)-1 {
            h.cursor++
            h.syncViewport()
        }
        return h, nil

    case tea.MouseButtonLeft:
        idx := h.listItemAt(msg.X, msg.Y)
        if idx < 0 || idx >= len(h.flatItems) {
            return h, nil
        }

        item := h.flatItems[idx]

        // Group row: single click toggles expand/collapse
        if item.Type == session.ItemTypeGroup {
            h.cursor = idx
            h.groupTree.ToggleGroup(item.Path)
            h.rebuildFlatItems()
            // Reposition cursor to the (possibly moved) group row
            for i, fi := range h.flatItems {
                if fi.Type == session.ItemTypeGroup && fi.Path == item.Path {
                    h.cursor = i
                    break
                }
            }
            h.saveGroupState()
            return h, nil
        }

        // Session row: check for double-click
        const doubleClickThreshold = 300 * time.Millisecond
        isDoubleClick := idx == h.lastClickIndex &&
            !h.lastClickTime.IsZero() &&
            time.Since(h.lastClickTime) < doubleClickThreshold

        // Always update tracking state
        h.lastClickIndex = idx
        h.lastClickTime = time.Now()

        // Move cursor to clicked row
        h.cursor = idx
        h.syncViewport()

        if isDoubleClick && item.Session != nil && item.Session.Exists() {
            h.isAttaching.Store(true)
            return h, h.attachSession(item.Session)
        }

        return h, nil
    }

    return h, nil
}
```

**Step 2: Wire into `Update()`**

In `Update()`, add before the `case tea.KeyMsg:` line (around line 3181):

```go
case tea.MouseMsg:
    return h.handleMouseMsg(msg)
```

**Step 3: Run tests**

```bash
go test ./internal/ui/... -run "TestHandleMouseMsg|TestListItemAt" -v
```
Expected: all tests PASS.

**Step 4: Run the full UI test suite**

```bash
go test ./internal/ui/... -v
```
Expected: all existing tests still pass.

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(ui): add mouse handler — click to select, double-click to attach, scroll to scroll"
```

---

### Task 7: Wire `EnableMouseMode()` in `attachSession()`

**Files:**
- Modify: `internal/ui/home.go:5237` (attachSession function)

**Context:** `EnableMouseMode()` is at `internal/tmux/tmux.go:1046` — it runs `tmux set-option -t <session> mouse on`. It's called on `*tmux.Session` (returned by `inst.GetTmuxSession()`). The user config is accessed via `session.LoadUserConfig()`. The call should happen after `tmuxSess.EnsureConfigured()` and before the `tea.Exec` call.

**Step 1: Write a test**

In `internal/ui/home_test.go`, add a focused note test (we can't easily test the side-effect without real tmux, so we verify the config accessor works and document the integration point):

```go
func TestAttachSessionMouseModeConfig(t *testing.T) {
    // Verify that GetMouseMode returns the expected value under various config states.
    // The integration (EnableMouseMode called on attach) is covered by manual testing.
    s := session.TmuxSettings{}
    if s.GetMouseMode() != false {
        t.Error("default MouseMode should be false")
    }
    tr := true
    s.MouseMode = &tr
    if !s.GetMouseMode() {
        t.Error("MouseMode should be true when set")
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/session/... -run TestAttachSessionMouseModeConfig -v
```
Expected: PASS (already covered by Task 1 logic, just a sanity check).

**Step 3: Modify `attachSession()`**

In `home.go`, in `attachSession()`, after the `tmuxSess.EnsureConfigured()` call (around line 5243), add:

```go
// Enable tmux mouse mode if configured (allows clicking status bar to switch windows)
cfg := session.LoadUserConfig()
if cfg.Tmux.GetMouseMode() {
    if err := tmuxSess.EnableMouseMode(); err != nil {
        uiLog.Warn("enable_mouse_mode_failed", slog.Any("error", err))
    }
}
```

**Step 4: Build to verify**

```bash
go build ./...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): enable tmux mouse mode on attach when mouse_mode = true"
```

---

### Task 8: Update config documentation

**Files:**
- Modify: `internal/session/userconfig.go:704-711` (TmuxSettings godoc example)

**Step 1: Update the example comment block**

Find the comment above `TmuxSettings` (lines 704-711) and add `mouse_mode`:

```go
// TmuxSettings allows users to override tmux options applied to every session.
// Options are applied AFTER agent-deck's defaults, so they take precedence.
//
// Example config.toml:
//
//	[tmux]
//	inject_status_line = false
//	mouse_mode = true
//	options = { "allow-passthrough" = "all", "history-limit" = "50000" }
```

**Step 2: Build and test one final time**

```bash
go build ./... && go test ./...
```
Expected: all pass.

**Step 3: Commit**

```bash
git add internal/session/userconfig.go
git commit -m "docs: document mouse_mode in TmuxSettings config example"
```

---

## Testing Checklist (manual)

After all tasks complete, verify end-to-end:

1. **Single click** — click any session row: cursor jumps to that row without attaching
2. **Double click** — double-click a running session: terminal hands off to tmux attach
3. **Group toggle** — click a group header: it expands/collapses
4. **Scroll wheel** — scroll up/down: list scrolls correctly with cursor tracking
5. **Right panel** — clicking in the preview pane (dual layout): no effect on list cursor
6. **Mouse mode config** — add `[tmux]\nmouse_mode = true` to `~/.agent-deck/config.toml`, attach a session, verify `tmux list-sessions` mouse option is on; verify clicking tmux status bar tabs switches windows
7. **Mouse mode off** — remove or set `mouse_mode = false`, attach session, verify tmux mouse is not set
