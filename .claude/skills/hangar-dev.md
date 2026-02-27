---
name: hangar-dev
description: >
  Use when working ON the Hangar codebase — fixing bugs, adding features, refactoring.
  Covers architecture orientation, known pitfalls, testing approach, and code conventions.
---

# Working on the Hangar Codebase

You are working on **Hangar**, a terminal session manager for AI coding agents. It is a Bubble Tea TUI on top of tmux, written in Go.

## Orientation Checklist

Before starting any change:

1. [ ] Run `go build ./...` — confirm clean baseline
2. [ ] Run `go test ./... 2>&1 | grep -v "PASS"` — note pre-existing failures only (see below)
3. [ ] Run `go test -race ./...` — the race detector should pass clean
4. [ ] Identify which package(s) your change touches (see Package Map below)

### Pre-Existing Failing Tests (do not fix without deliberate effort)
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

---

## Package Map

```
cmd/hangar/           CLI commands (19 subcommands)
  main.go             handleAdd() — session creation entry point
  session_cmd.go      12 session subcommands
  worktree_cmd.go     worktree list/info/cleanup/finish
  launch_cmd.go       combined add+start+send
  hook_handler.go     processes Claude Code hook events → writes ~/.hangar/hooks/*.json

internal/session/     Core domain (start here for session logic)
  instance.go         Session struct + GetStatus/SetStatus + AI tool integration
  storage.go          SQLite CRUD for sessions and groups
  groups.go           GroupTree — hierarchical session grouping
  hook_watcher.go     fsnotify watcher for ~/.hangar/hooks/ (fast-path status)
  event_watcher.go    fsnotify watcher for general events
  transition_daemon.go  Background adaptive-polling daemon
  transition_notifier.go  Parent-child session notification delivery
  notifications.go    Notification bar state (waiting sessions)
  todo.go             Todo domain model

internal/tmux/        Zero-subprocess tmux interaction
  pipemanager.go      Persistent tmux -C control pipes, reconnect logic
  controlpipe.go      Protocol handler for tmux control mode
  detector.go         Claude/Gemini/shell status from spinner/prompt patterns
  pty.go              PTY attachment with SIGWINCH, terminal restore

internal/ui/          Bubble Tea TUI
  home.go             10,322-line main model (see below)
  styles.go           All lipgloss styles — define here, not inline
  [dialog].go         Each dialog is a separate file (newdialog, diff_view, etc.)

internal/statedb/     SQLite wrapper
  statedb.go          schema, WAL mode, migrations, heartbeat election
```

---

## Working in home.go (the Big File)

`home.go` is 10,322 lines with 149 functions. Navigate it by function name — your editor's symbol search is your friend.

### The 7-Place Dialog Wiring Pattern

Every dialog requires changes in **7 places** inside `home.go`. When adding a new dialog:

1. **Struct field** — add to `Home` struct
2. **Init** — initialize in `NewHomeWithProfileAndMode()`
3. **SetSize** — call `dialog.SetSize()` in `Update(tea.WindowSizeMsg)`
4. **View guard** — `if h.dialog.IsVisible() { return h.dialog.View() }` in `View()`
5. **Key routing** — in `handleMainKey()`, check `h.dialog.IsVisible()` first
6. **Mouse guard** — in `handleMouseMsg()`, bail if any dialog is open
7. **Action handler** — process dialog results in `Update()` switch

Miss any one and you get mysterious event leaks or invisible dialogs.

### Update() Navigation

The `Update()` method (lines 2167–3593) dispatches on ~45 message types. The critical path for key events:

```
tea.KeyMsg
  → check if any dialog active → route to that dialog's HandleKey()
  → if no dialog → handleMainKey()
      → navigation keys (j/k/↑/↓/g/G)
      → action keys (n=new, d=delete, D=diff, W=worktree, etc.)
      → escape handling (double-escape=quit)
```

### Rendering Performance Rules

1. **Never call `lipgloss.NewStyle()` inside a render function.** Define styles at package level in `styles.go`. This is the #1 allocation hotspot.
2. **Re-use `h.viewBuilder`** (a `strings.Builder` field) instead of allocating local builders in `View()`.
3. `renderPreviewPane()` is 1,033 lines — it runs every frame. Be conservative adding to it.
4. The status update round-robin (5–10 sessions/tick) is intentional — do not change to "update all".

---

## Concurrency Rules

### The instancesMu Contract

`h.instances` (slice) and `h.instanceByID` (map) are **always** updated together under `h.instancesMu`. Never modify one without the other. Never unlock between the two operations.

```go
// CORRECT
h.instancesMu.Lock()
h.instances = append(h.instances[:i], h.instances[i+1:]...)
delete(h.instanceByID, id)
h.instancesMu.Unlock()

// WRONG — currently exists as a bug at sessionDeletedMsg handler
h.instancesMu.Unlock()
delete(h.instanceByID, id)  // race!
```

### Storage Lifecycle

`session.Storage` holds a database connection and **must be closed**. Use `defer storage.Close()` immediately after `NewStorageWithProfile()`. Failing to do so accumulates file handles (low-severity leak but real).

### Hook/Event Watcher Debounce

Both `hook_watcher.go` and `event_watcher.go` use a `time.AfterFunc` debounce pattern. The timer must be stopped (`debounceTimer.Stop()`) before the watcher goroutine exits, or the callback may fire on a cancelled context.

---

## Known Bugs (from deep-dive 2026-02-26)

| Severity | Location | Bug |
|----------|----------|-----|
| Medium | `ui/home.go` sessionDeletedMsg | `delete(instanceByID)` outside mutex |
| Medium | `session/transition_notifier.go:123` | Storage opened, never closed |
| Medium | `cmd/hangar/session_cmd.go:879` | `session set path` accepts any string |
| Low | `session/hook_watcher.go` | debounce timer not stopped on shutdown |
| Low | `session/event_watcher.go` | same debounce timer issue |
| Low | `session/groups.go:172` | "conductor" magic string with hardcoded Order=-1 |
| Low | `statedb/statedb.go:236` | string-based error matching on migration |

---

## Testing Approach

### What's tested well
- Hook handler (`hook_handler_test.go`) — all event types, atomic write, missing env var
- CLI utilities (`cli_utils_test.go`) — flag reordering, session resolution, command parsing
- Session storage — CRUD, migration from JSON
- tmux patterns — spinner detection, prompt matching, ANSI stripping
- Dialogs — most have companion test files

### What lacks tests (don't add features here without adding tests first)
- `launch_cmd.go` — no test file
- `project_cmd.go` — no test file
- `notify_daemon_cmd.go` — no test file
- `handleAdd()` — no integration test (only unit tests for helpers)
- `handleWorktreeFinish()` — merge conflict scenarios untested
- Concurrent hook writes — race not tested

### Test Patterns

```go
// UI tests use home_test.go helpers — check testmain_test.go for setup
// Session tests use session/testmain_test.go with temp SQLite DB
// Run specific test:
go test ./internal/session/... -run TestGroupTree -v
// Run with race:
go test -race ./internal/tmux/...
```

---

## Adding a New AI Tool Integration

Instance.go has tool-specific branches for Claude, Gemini, OpenCode, Codex, shell. To add a new tool:

1. Add detection in `session/tooloptions.go` — `detectTool()`
2. Add session-ID extraction in `instance.go` — `GetToolSessionID()` pattern
3. Add command builder in `instance.go` — `buildXxxCommand()`
4. Add analytics in `session/analytics.go` if needed
5. Update `detector.go` prompt/spinner patterns for the new tool
6. Update `cmd/hangar/cli_utils.go` `resolveSessionCommand()`
7. Update badge rendering in `home.go` `renderSessionItem()`

---

## Refactoring Guidelines

When reducing home.go, extract by **rendering concern** not by feature:

- **Rendering functions** can move to new files — they only need the `Home` struct as receiver
- **Message handlers** should stay in home.go (they modify state, need full visibility)
- **Business logic** that doesn't touch `h.` fields belongs in the appropriate `internal/` package
- New files in `internal/ui/` must use the same package name (`package ui`)

---

## Useful Commands

```bash
# Find all dialog struct fields in Home
grep -n "Dialog\|dialog\|Overlay\|overlay" internal/ui/home.go | grep "^[0-9]*:[[:space:]]*[a-z]"

# Find all message types handled in Update()
grep -n "^[[:space:]]*case " internal/ui/home.go | head -60

# Check goroutine count in tests
go test -v ./internal/tmux/... 2>&1 | grep -i goroutine

# Profile allocations
go test -bench=. -memprofile=mem.prof ./internal/ui/...
go tool pprof mem.prof
```
