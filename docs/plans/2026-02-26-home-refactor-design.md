# Design: home.go Refactor, Bug Fixes & Performance

**Date**: 2026-02-26
**Scope**: `internal/ui/home.go` restructure, 4 bug fixes, rendering performance
**Approach**: Outside-in (Approach A) — always green between phases

---

## Problem Statement

After a full codebase deep-dive (2026-02-26), three problem areas were identified:

1. **home.go monolith**: 10,322 lines, `Update()` at 1,400+ lines, `renderPreviewPane()` at 1,033 lines, 7-place dialog wiring pattern, 5 independent caches with scattered TTL logic
2. **Performance**: `lipgloss.NewStyle()` called inside per-line render loops (allocations every frame), no memoization of preview sections
3. **Bugs**: 4 confirmed bugs including a mutex gap, file handle leak, missing input validation, and debounce timer cleanup issue

---

## Design Constraints

- Each phase must leave all existing passing tests green
- Existing pre-existing failing tests (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`) remain acknowledged but not addressed here
- Add targeted snapshot/behavior tests before moving any code
- No public API changes (all internal packages)

---

## Phase 1: Bug Fixes

### 1a — `instanceByID` Mutex Gap

**Location**: `internal/ui/home.go`, `sessionDeletedMsg` handler
**Bug**: `delete(h.instanceByID, msg.deletedID)` executes after `h.instancesMu.Unlock()`, creating a window where the slice and map are inconsistent.

**Fix**: Extract a `removeInstance(id string)` helper:
```go
func (h *Home) removeInstance(id string) {
    h.instancesMu.Lock()
    defer h.instancesMu.Unlock()
    for i, s := range h.instances {
        if s.ID == id {
            h.instances = append(h.instances[:i], h.instances[i+1:]...)
            break
        }
    }
    delete(h.instanceByID, id)
}
```
Replace the two operations in `sessionDeletedMsg` with `h.removeInstance(msg.deletedID)`.

**Test**: Add `TestRemoveInstance_MapAndSliceConsistent` — create a Home with 3 instances, remove the middle one, assert both `instances` and `instanceByID` are consistent.

---

### 1b — Storage Leak in `dispatch()`

**Location**: `internal/session/transition_notifier.go:123`
**Bug**: `NewStorageWithProfile()` opens a SQLite connection that is never closed — file handles accumulate until GC.

**Fix**: Add `defer storage.Close()` immediately after the successful open:
```go
storage, err := NewStorageWithProfile(event.Profile)
if err != nil { ... }
defer storage.Close()
```

**Test**: Add `TestDispatch_ClosesStorage` — mock or instrument storage to verify Close() is called.

---

### 1c — Path Validation in `session set`

**Location**: `cmd/hangar/session_cmd.go:879`
**Bug**: `session set <id> path <value>` stores any string without validation — can create sessions pointing at non-existent or non-directory paths.

**Fix**: In the `case "path":` branch, validate before storing:
```go
case "path":
    info, err := os.Stat(value)
    if err != nil {
        out.Error(fmt.Sprintf("path does not exist: %s", value), ErrCodeInvalidOperation)
        os.Exit(1)
    }
    if !info.IsDir() {
        out.Error(fmt.Sprintf("path is not a directory: %s", value), ErrCodeInvalidOperation)
        os.Exit(1)
    }
    oldValue = inst.ProjectPath
    inst.ProjectPath = value
```

**Test**: Add `TestSessionSetPath_Validation` — attempt to set path to `/nonexistent`, assert error exit.

---

### 1d — Debounce Timer Cleanup

**Location**: `internal/session/hook_watcher.go`, `internal/session/event_watcher.go`
**Bug**: `time.AfterFunc` debounce timer may fire after context cancellation if not stopped.

**Fix**: In the cleanup path of each watcher's goroutine, stop the timer:
```go
// In cleanup before returning:
w.mu.Lock()
if t := w.debounceTimer; t != nil {
    t.Stop()
}
w.mu.Unlock()
```

**Test**: Existing watcher tests should still pass; add a test that cancels the watcher mid-debounce and verifies no callback fires.

---

## Phase 2: Renderer Extraction

Extract rendering-only functions from `home.go` into three new files. These are pure string-producing functions — no state mutations, safe to move.

### Before moving each function: add a snapshot test

```go
func TestRenderPreviewPane_Smoke(t *testing.T) {
    h := newTestHome(t)
    // set up minimal session state
    result := h.renderPreviewPane(80, 40)
    if result == "" {
        t.Fatal("expected non-empty preview")
    }
    // assert key strings present (not exact snapshot — too brittle)
}
```

### New file: `internal/ui/preview_renderer.go`

Move from `home.go`:
- `renderPreviewPane(width, height int) string` — split internally into 5 helpers:
  - `renderPreviewHeader(s *session.Instance, width int) string`
  - `renderPreviewInfoSection(s *session.Instance, width, height int) string`
  - `renderPreviewDiffSection(s *session.Instance, width, height int) string`
  - `renderPreviewOutputSection(s *session.Instance, width, height int) string`
  - `renderPreviewFooter(width int) string`
- `renderGroupPreview(g *session.Group, width, height int) string`
- `renderSessionInfoCard(s *session.Instance, width int) string`

**LOC reduction in home.go**: ~1,100 lines

### New file: `internal/ui/list_renderer.go`

Move from `home.go`:
- `renderSessionList(width, height int) string`
- `renderSessionItem(item session.Item, index int, width int) string`
- `renderGroupItem(item session.Item, index int, width int) string`

**LOC reduction in home.go**: ~250 lines

### New file: `internal/ui/layout_renderer.go`

Move from `home.go`:
- `renderDualColumnLayout(width, height int) string`
- `renderStackedLayout(width, height int) string`
- `renderSingleColumnLayout(width, height int) string`

**LOC reduction in home.go**: ~150 lines

### Expected home.go size after Phase 2

~8,800 lines (down ~1,500 from 10,322)

---

## Phase 3: Reshape `Update()`

### New file: `internal/ui/update_handlers.go`

All message cases with more than ~20 lines of logic become methods:

```go
// update_handlers.go
func (h *Home) handleLoadSessions(msg loadSessionsMsg) tea.Cmd
func (h *Home) handleStorageChanged(msg storageChangedMsg) tea.Cmd
func (h *Home) handleSessionCreated(msg sessionCreatedMsg) tea.Cmd
func (h *Home) handleSessionDeleted(msg sessionDeletedMsg) tea.Cmd
func (h *Home) handleSessionForked(msg sessionForkedMsg) tea.Cmd
func (h *Home) handleWorktreeCreated(msg worktreeCreatedForNewSessionMsg) tea.Cmd
func (h *Home) handleStatusUpdate(msg statusUpdateMsg) tea.Cmd
func (h *Home) handlePreviewFetched(msg previewFetchedMsg) tea.Cmd
func (h *Home) handleDiffFetched(msg diffFetchedMsg) tea.Cmd
func (h *Home) handleUpdateCheck(msg updateCheckMsg) tea.Cmd
func (h *Home) handleMouseMsg(msg tea.MouseMsg) tea.Cmd   // already exists, keep
```

### `Update()` after Phase 3

```go
func (h *Home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:      h.width = msg.Width; h.height = msg.Height; ...
    case loadSessionsMsg:        return h, h.handleLoadSessions(msg)
    case storageChangedMsg:      return h, h.handleStorageChanged(msg)
    case sessionCreatedMsg:      return h, h.handleSessionCreated(msg)
    case sessionDeletedMsg:      return h, h.handleSessionDeleted(msg)
    case sessionForkedMsg:       return h, h.handleSessionForked(msg)
    case worktreeCreatedForNewSessionMsg: return h, h.handleWorktreeCreated(msg)
    case statusUpdateMsg:        return h, h.handleStatusUpdate(msg)
    case previewFetchedMsg:      return h, h.handlePreviewFetched(msg)
    case diffFetchedMsg:         return h, h.handleDiffFetched(msg)
    case updateCheckMsg:         return h, h.handleUpdateCheck(msg)
    case tea.MouseMsg:           return h, h.handleMouseMsg(msg)
    case tea.KeyMsg:             return h, h.handleKeyMsg(msg)
    }
    return h, nil
}
```

**LOC reduction in home.go**: ~800 lines moved to `update_handlers.go`
**Expected home.go size after Phase 3**: ~8,000 lines

---

## Phase 4: Dialog Manager

### New interface: `internal/ui/dialog.go`

```go
// Dialog is the interface all dialogs must implement.
type Dialog interface {
    IsVisible() bool
    Show()
    Hide()
    View() string
    HandleKey(key tea.KeyMsg) (cmd tea.Cmd, consumed bool)
    SetSize(width, height int)
}
```

Most existing dialogs already have these methods — a rename pass makes them match.

### New type: `internal/ui/dialog_manager.go`

```go
type DialogManager struct {
    stack []Dialog
    mu    sync.Mutex
}

func (dm *DialogManager) Push(d Dialog)
func (dm *DialogManager) Pop()
func (dm *DialogManager) Active() Dialog   // nil if empty
func (dm *DialogManager) IsAnyVisible() bool
func (dm *DialogManager) View() string     // delegates to Active(), "" if none
func (dm *DialogManager) HandleKey(key tea.KeyMsg) (tea.Cmd, bool)
func (dm *DialogManager) SetSize(w, h int) // broadcasts to all registered dialogs
func (dm *DialogManager) Register(d Dialog) // for SetSize broadcast
```

### Changes in `home.go`

**View()**: Replace 17-line visibility check cascade with:
```go
if v := h.dialogs.View(); v != "" {
    return h.ensureExactHeight(v)
}
```

**Key routing** in `handleKeyMsg()`: Replace per-dialog `IsVisible()` checks with:
```go
if cmd, consumed := h.dialogs.HandleKey(msg); consumed {
    return cmd
}
```

**Mouse guard** in `handleMouseMsg()`:
```go
if h.dialogs.IsAnyVisible() {
    return nil
}
```

**SetSize** in `Update(tea.WindowSizeMsg)`: Replace 15+ individual `dialog.SetSize()` calls with:
```go
h.dialogs.SetSize(msg.Width, msg.Height)
```

**Adding new dialogs** (reduced from 7 places to 3):
1. Create dialog struct implementing `Dialog`
2. Register: `h.dialogs.Register(myDialog)` in `NewHomeWithProfileAndMode()`
3. Push on trigger: `h.dialogs.Push(h.myDialog)` at the key handler
4. Handle result in `update_handlers.go`

**LOC reduction in home.go**: ~300 lines (the 17-line cascade + scattered per-dialog routing)

---

## Phase 5: Cache Consolidation

### New type: `internal/ui/ui_cache.go`

```go
type UICache struct {
    mu      sync.RWMutex
    entries map[string]map[string]cacheEntry // [sessionID][key] -> entry
}

type cacheEntry struct {
    value   any
    cachedAt time.Time
    ttl     time.Duration
}

// TTL constants
const (
    previewTTL      = 30 * time.Second
    prInfoTTL       = 60 * time.Second
    worktreeDirtyTTL = 5 * time.Minute
)

// Typed accessors
func (c *UICache) GetPreview(id string) (string, bool)
func (c *UICache) SetPreview(id string, v string)
func (c *UICache) GetPRInfo(id string) (*prInfo, bool)
func (c *UICache) SetPRInfo(id string, v *prInfo)
func (c *UICache) GetWorktreeDirty(id string) (bool, bool)
func (c *UICache) SetWorktreeDirty(id string, v bool)

// Invalidation
func (c *UICache) InvalidateSession(id string)
func (c *UICache) InvalidateAll()
func (c *UICache) Prune()  // evict expired entries
```

**In `home.go`**: Replace 5 separate cache maps + 5 mutexes + scattered TTL checks with:
```go
cache *UICache  // single field
```

**LOC reduction in home.go**: ~200 lines (cache fields, mutex declarations, TTL checks)

---

## Phase 6: Performance

### 6a — Module-Level Styles (required)

Audit `preview_renderer.go`, `list_renderer.go`, and `layout_renderer.go` (newly extracted in Phase 2) for any `lipgloss.NewStyle()` calls inside functions. Move all to package-level `var` declarations in `styles.go`.

Pattern to eliminate:
```go
// BEFORE — allocates every call
lipgloss.NewStyle().Foreground(ColorText).Render(line)

// AFTER — style declared once at package level
var styleBodyText = lipgloss.NewStyle().Foreground(ColorText)
// then:
styleBodyText.Render(line)
```

All styles must be in `styles.go`. This is both a performance fix and a consistency rule.

### 6b — Preview Section Memoization (recommended)

Using the `UICache` from Phase 5, memoize each preview section independently. Each section key includes the session ID + a version token:

```go
// In renderPreviewInfoSection:
key := fmt.Sprintf("preview-info-%s-%d", s.ID, s.lastActivityVersion)
if cached, ok := h.cache.GetPreview(key); ok {
    return cached
}
result := ... // render
h.cache.SetPreview(key) // stored with 30s TTL
return result
```

Version tokens:
- Info section: `s.LastActivity.Unix()`
- Diff section: `s.DiffStats` hash or 60s TTL
- Output section: `s.LastActivity.Unix()` with 5s TTL
- Header: `string(s.Status)` + `s.Title`
- Footer: static per `(width, height)` pair

### 6c — Builder Capacity Hints

In `renderSessionList()`, pre-allocate the builder:
```go
b.Grow(len(h.flatItems) * 80)  // ~80 bytes per row estimate
```

---

## Expected Outcomes

| Metric | Before | After |
|--------|--------|-------|
| `home.go` lines | 10,322 | ~6,500 |
| `Update()` lines | ~1,400 | ~80 |
| `renderPreviewPane()` lines | 1,033 | ~150 (+ 5 helpers ~80ln each) |
| Dialog wiring places | 7 | 3 |
| Cache mutexes | 5 | 1 |
| Frame allocations (preview) | O(n lines) | ~0 (memoized) |

---

## File Map After Completion

```
internal/ui/
  home.go               (~6,500 ln) — struct, Init, Update dispatch, View dispatch, key handlers
  update_handlers.go    (~800 ln)   — one handler per message type
  dialog.go             (~30 ln)    — Dialog interface
  dialog_manager.go     (~120 ln)   — DialogManager
  ui_cache.go           (~150 ln)   — UICache with typed accessors
  preview_renderer.go   (~500 ln)   — renderPreviewPane + 5 section helpers
  list_renderer.go      (~250 ln)   — renderSessionList, renderSessionItem, renderGroupItem
  layout_renderer.go    (~150 ln)   — responsive layout dispatch
  styles.go             (~900 ln)   — all lipgloss styles (expanded with moved styles)
  [15 existing dialog files, unchanged]
```

---

## Risk Mitigation

- **Run `go test -race ./internal/ui/...` after every phase** before committing
- Phase 2 (extraction) carries the highest mechanical risk — use snapshot tests before moving
- Phase 4 (dialog manager) touches key routing — test each dialog open/close/escape path manually after
- Phase 5 (cache) is pure refactor with no behavior change — the cache tests validate correctness
- Phase 6a (styles) is low-risk but scan for any styles that depend on runtime state (color overrides, theme) — those may need to be functions, not vars

---

## Phases Summary

| Phase | What | Risk | Est. LOC Δ in home.go |
|-------|------|------|----------------------|
| 1 | Bug fixes (4) | Low | ~0 |
| 2 | Extract renderers | Medium | -1,500 |
| 3 | Reshape Update() | Medium | -800 |
| 4 | Dialog manager | Medium-High | -300 |
| 5 | Cache consolidation | Medium | -200 |
| 6 | Performance | Low-Medium | ~0 (moves to styles.go) |
| **Total** | | | **~-2,800 from home.go** |
