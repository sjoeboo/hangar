# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**Hangar** is a terminal session manager for AI coding agents, built as a TUI (Terminal User Interface) on top of tmux. It manages Claude Code (and other AI tool) sessions, providing status monitoring, worktree management, project organization, and lifecycle hooks.

- **Binary**: `hangar`
- **Module**: `github.com/sjoeboo/hangar`
- **Config dir**: `~/.hangar/`
- **Tmux session prefix**: `hangar_`

## Architecture

```
cmd/hangar/          # CLI entry point and subcommands (~19 subcommands)
internal/
  session/           # Core session model, config, hooks, status detection, todos
  statedb/           # SQLite state (WAL mode, heartbeat-based primary election)
  tmux/              # tmux interaction — zero-subprocess control mode architecture
  ui/                # Bubble Tea TUI (home.go is 10,322 lines; see extraction plan)
  git/               # Git worktree operations + diff fetching
  update/            # Self-update logic
  profile/           # Multi-profile support
  logging/           # Structured logging with ring buffer + aggregator
  clipboard/         # Cross-platform clipboard
  experiments/       # Simple feature flag system
  platform/          # WSL/macOS/Linux detection, fsnotify support check
```

### Key Files

| File | Lines | Purpose |
|------|-------|---------|
| `internal/ui/home.go` | 10,322 | Main TUI model — all key bindings, view rendering, state |
| `internal/session/instance.go` | 4,131 | Session data model + AI tool integration + status detection |
| `internal/session/groups.go` | 1,105 | Hierarchical group tree with flatten-for-TUI |
| `internal/ui/newdialog.go` | 1,006 | New session creation dialog with worktree/MCP selection |
| `internal/ui/styles.go` | 719 | Color palette, lipgloss styles, theme (oasis_lagoon_dark) |
| `internal/session/storage.go` | 678 | SQLite-backed session persistence |
| `internal/session/transition_daemon.go` | 395 | Adaptive polling background daemon |
| `internal/ui/diff_view.go` | 363 | Inline diff overlay (D key) |
| `internal/ui/todo_dialog.go` | 875 | Kanban todo board (t key) |
| `internal/tmux/pipemanager.go` | 421 | tmux control pipe lifecycle + reconnect |
| `internal/tmux/detector.go` | 539 | Status detection via prompt/spinner patterns |
| `internal/statedb/statedb.go` | 773 | SQLite wrapper with schema migrations |
| `cmd/hangar/main.go` | ~2700 | CLI dispatch and handleAdd |
| `cmd/hangar/session_cmd.go` | ~2000 | 12 session subcommands |
| `internal/session/claude_hooks.go` | — | Claude Code lifecycle hook injection/removal |
| `internal/session/todo.go` | 178 | Todo domain model, Storage CRUD |
| `internal/session/config.go` | — | `GetHangarDir()` — base config directory |

## Build & Test

```bash
go build ./...                  # Build
go test ./...                   # All tests
go test -race ./...             # Tests with race detector (passes clean)
go test ./internal/ui/...       # UI tests only
go test ./internal/session/...  # Session tests
go run ./cmd/hangar              # Run locally
```

**Pre-existing failing tests** (do not fix without understanding root cause):
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

## UI Architecture (Bubble Tea)

Hangar uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. Key patterns:

- **State changes** go through `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — never mutate state in `View()`
- **Async work** returns `tea.Cmd` functions that produce `tea.Msg` results dispatched back to `Update()`
- **Dialogs** are structs with `IsVisible()`, `Show()`, `Hide()`, `View()`, `HandleKey()`, `SetSize()` methods
- **Adding a new dialog** requires wiring in 7 places in `home.go`: struct field, init, key routing, mouse guard, trigger key, SetSize, View check

### home.go Internal Structure

`Update()` is 1,400+ lines with 45+ message type cases. Key message flow:
```
tea.KeyMsg → handleMainKey() → handleXxxDialogKey() / direct state mutation
loadSessionsMsg → rebuilds instances + groupTree + flatItems
storageChangedMsg → triggers reload with debounce
statusUpdateMsg → round-robin background status update (5-10 sessions/tick)
```

**Rendering pipeline** (called every frame):
```
View() → layout dispatch (single/stacked/dual column)
       → renderPreviewPane() [1,033 lines — main bottleneck]
       → renderSessionList() → renderSessionItem() per visible item
```

**Performance optimizations already in place:**
- Round-robin status updates (5-10 sessions/tick) — 90% CPU reduction
- `viewBuilder` strings.Builder reuse
- Preview cache per session (30s TTL)
- Worktree dirty cache (5m TTL)
- PR cache (60s TTL)

### Extraction Plan for home.go (future work)

```
home.go (~10k lines) should be split into:
  home.go               (~4,000) — core model, Init/Update/View routing
  sessionListRenderer.go (~400)  — renderSessionList, renderSessionItem
  previewRenderer.go    (~500)   — renderPreviewPane split into sections
  layoutRenderer.go     (~150)   — responsive layout dispatch
  statusManager.go      (~150)   — status counting, caching, filtering
```

## Session Status Detection

Status is detected via two parallel systems:

1. **Hook-based** (fast path): Claude Code writes JSON to `~/.hangar/hooks/{id}.json` via lifecycle hooks. Detected via `fsnotify` with 100ms debounce.
2. **Tmux-based** (fallback): Periodic polling via control-mode pipe — reads pane title (spinner chars + prompt patterns), activity timestamps, and content scanning. **Zero subprocess** architecture — uses `tmux -C` control mode kept open.

Hook command written to Claude's `settings.json`: `hangar hook-handler`

### Status Detection Internals

```
hookWatcher (fsnotify) ─────────────────────────┐
                                                 ▼
tmux controlpipe ──→ pipemanager ──→ detector ──→ instance.SetStatus()
                      (reconnect)    (patterns)        ▲
                                                 │
eventWatcher (fsnotify) ─────────────────────────┘
```

## Key Bindings (TUI)

| Key | Action |
|-----|--------|
| `enter` | Attach to session |
| `n` | New session |
| `p` | New project/group |
| `x` | Send message to session |
| `M` | Move session to group |
| `r` | Rename session/group |
| `R` | Restart session |
| `W` | Worktree finish (archive branch) |
| `D` | Inline diff view overlay (worktree sessions) |
| `P` | PR overview — all sessions with PRs |
| `t` | Todo kanban board (per-project) |
| `d` | Delete session/group |
| `o` | Open PR in browser |
| `G` | Open lazygit |
| `S` | Settings |
| `/` | Search |
| `~` | Toggle status sort |
| `Ctrl+R` | Force-refresh git/PR status for selected session |
| `?` | Help overlay |
| `Ctrl+Q` | Detach from session |
| `q` | Quit |

## View Modes

`home.go` uses a `viewMode` field to switch between top-level views:

| `viewMode` | Description |
|------------|-------------|
| `""` (default) | Main session list |
| `"prs"` | PR overview (P key) — shows all sessions with PRs |

The diff overlay (`DiffView`) and todo dialog (`TodoDialog`) are separate overlays rendered on top of the main view, not separate view modes.

## Claude Code Hooks

Hangar installs hooks into Claude Code's `~/.claude/settings.json` to detect session lifecycle events (SessionStart, Stop, UserPromptSubmit, PermissionRequest, etc.). The hook command is `hangar hook-handler`.

To install hooks: `hangar hooks install`
To check status: `hangar hooks status`

Hook handler writes atomic JSON status files (`tmpfile + rename`). Hook files are stored at `~/.hangar/hooks/{id}.json` and are world-readable (0644) — they contain only status info, no secrets.

HTTP hooks (Claude >= 2.1.63) are enabled by default on port 2437. To disable HTTP hooks
and use the command hook fallback only, set `hookServerPort: 0` in `~/.hangar/config.yaml`.

## Style / Conventions

- **Theme**: oasis_lagoon_dark (defined in `internal/ui/styles.go`)
- **Error display**: Use `h.setError(err)` for transient status bar messages (auto-dismiss after ~3s)
- **Async patterns**: Background goroutines return typed `tea.Msg` structs — see `sendTextResultMsg`, `worktreeFinishResultMsg`, `diffFetchedMsg`, `worktreeCreatedForNewSessionMsg` for examples
- **Logging**: `slog` throughout; log files in `~/.hangar/logs/`
- **PR cache**: `prCache map[string]*prInfo` protected by `prCacheMu sync.Mutex`; 60-second TTL; populated by `gh pr view --json`
- **Todo storage**: SQLite via `internal/statedb`; `session/todo.go` provides domain model + Storage methods with mutex guards
- **Sidebar badges**: Tool badge is hidden for `claude` (default); only shown for non-default tools. Worktree branch badge removed from sidebar (shown in detail panel instead)
- **Worktree lifecycle**: Never offer to merge worktrees directly into master/main — PRs are the product of a session. Worktree cleanup (`W` key) archives the branch, not merges it.
- **Worktree creation**: Always update the base branch (main/master) first before branching a new worktree.
- **Complex tasks**: Use agent teams (`TeamCreate`) when possible for multi-step work; fall back to parallel subagents via `Task`.
- **Styles**: Pre-compile `lipgloss.Style` at module level, not inside render functions — creating styles in hot loops causes measurable allocation overhead.
- **Mutex discipline**: Always hold `instancesMu` for the full operation when modifying both `instances` slice and `instanceByID` map together. Don't unlock between slice and map operations.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `HANGAR_INSTANCE_ID` | Current session ID |
| `HANGAR_TITLE` | Current session title |
| `HANGAR_TOOL` | Tool in use (claude, shell, etc.) |
| `HANGAR_PROFILE` | Active profile |

## Known Bugs & Issues (from deep-dive 2026-02-26)

### Medium Priority

1. **instanceByID can diverge from instances slice** (`internal/ui/home.go`)
   - `delete(h.instanceByID, msg.deletedID)` happens outside the `instancesMu.Lock()` block in `sessionDeletedMsg` handler
   - Fix: wrap both slice removal AND map delete in a single lock/unlock

2. **Storage not closed in dispatch()** (`internal/session/transition_notifier.go:123`)
   - `dispatch()` opens `NewStorageWithProfile()` but never calls `defer storage.Close()`
   - File handles accumulate until GC; fix with `defer storage.Close()`

3. **session set path field not validated** (`cmd/hangar/session_cmd.go:879`)
   - `session set <id> path <value>` stores any string without `os.Stat()` validation
   - Fix: validate path exists before storing

4. **Debounce timer not stopped on shutdown** (`internal/session/hook_watcher.go`, `event_watcher.go`)
   - `debounceTimer` may fire after context cancellation; add `debounceTimer.Stop()` in cleanup

### Low Priority

5. **Session name parsing breaks with underscores in title**
   - `session_cmd.go`: uses `strings.LastIndex("_")` to split `hangar_<title>_<id>`
   - Titles with underscores produce wrong title extraction (ID is UUID suffix so this usually works, but it's fragile)

6. **String-based error matching in schema migration** (`internal/statedb/statedb.go:236`)
   - `strings.Contains(err.Error(), "duplicate column name")` is fragile across SQLite versions

7. **"conductor" magic string in groups** (`internal/session/groups.go:172`)
   - Gets special `Order = -1` treatment; if user creates group named "conductor" it re-orders unexpectedly

8. **pendingTitleChanges map may race** (`internal/ui/home.go:2285+`)
   - Iterated without mutex during reload; low severity as it's only written in Update()

## Test Coverage Gaps

Files with NO tests: `launch_cmd.go`, `project_cmd.go`, `notify_daemon_cmd.go`

Critical paths lacking integration tests:
- `handleAdd()` — end-to-end session creation
- `handleSession start/stop` — full lifecycle
- `handleLaunch()` — combined create + start + send
- `handleWorktreeFinish()` — merge + cleanup
- Concurrent hook writes

## tmux Layer Details

The tmux layer uses a **zero-subprocess control mode** architecture:
- `pipemanager.go`: Maintains persistent `tmux -C` processes per session; exponential backoff reconnect (2s→4s→8s→16s→30s max, 5 retries)
- `controlpipe.go`: Protocol handler for `%output`, `%begin`, `%end`, `%error` messages; 2MB buffer; 2s handshake timeout
- `detector.go`: 90+ Claude-specific prompt words; Braille + asterisk spinner detection; O(n) `StripANSI` (fixed from O(n²))
- `pty.go`: Full PTY with `SIGWINCH` propagation, WaitGroup-tracked goroutines, proper terminal state restore

## statedb Schema

6 core tables: `metadata`, `instances`, `groups`, `instance_heartbeats`, `todos`
- WAL mode for concurrent readers
- 5-second busy timeout for inter-process serialization
- Heartbeat-based primary election among multiple hangar instances
- Schema version in metadata table with ALTER TABLE migrations (v1→v4+)

## Dependencies (all current as of Feb 2026)

All dependencies are up-to-date. Key ones:
- `charmbracelet/bubbletea v1.3.10` — TUI framework
- `charmbracelet/lipgloss v1.1.0` — terminal styling
- `modernc.org/sqlite v1.44.3` — pure-Go SQLite (no CGO)
- `fsnotify v1.9.0` — file watching
- `golang.org/x/term v0.37.0` — terminal control

No known CVEs in any dependency.

## Release

Releases are built via GoReleaser (`go install github.com/goreleaser/goreleaser/v2@latest`):

```bash
goreleaser release --snapshot --clean   # Test release build
goreleaser release                       # Actual release (requires GITHUB_TOKEN)
```

Release artifacts are named `hangar_{version}_{os}_{arch}.tar.gz`.

Homebrew formula is at `sjoeboo/homebrew-tap`.
