# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**Hangar** is a terminal session manager for AI coding agents, built as a TUI (Terminal User Interface) on top of tmux. It manages Claude Code (and other AI tool) sessions, providing status monitoring, worktree management, project organization, and lifecycle hooks.

- **Binary**: `hangar`
- **Module**: `ghe.spotify.net/mnicholson/hangar`
- **Config dir**: `~/.hangar/`
- **Tmux session prefix**: `hangar_`

## Architecture

```
cmd/hangar/          # CLI entry point and subcommands
internal/
  session/           # Core session model, config, hooks, status detection
  tmux/              # tmux interaction layer (status, capture, send-keys)
  ui/                # Bubble Tea TUI (home.go is the main model ~8500 lines)
  git/               # Git worktree operations
  update/            # Self-update logic
  profile/           # Multi-profile support
  logging/           # Structured logging
```

### Key Files

| File | Purpose |
|------|---------|
| `internal/ui/home.go` | Main TUI model — all key bindings, view rendering, state |
| `internal/ui/styles.go` | Color palette, lipgloss styles, theme (oasis_lagoon_dark) |
| `internal/session/instance.go` | Session data model + status detection logic |
| `internal/session/claude_hooks.go` | Claude Code lifecycle hook injection/removal |
| `internal/tmux/tmux.go` | tmux session management, status detection, send-keys |
| `internal/session/config.go` | `GetHangarDir()` — base config directory |
| `cmd/hangar/main.go` | CLI dispatch and subcommands |

## Build & Test

```bash
go build ./...                  # Build
go test ./...                   # All tests
go test ./internal/ui/...       # UI tests only
go test ./internal/session/...  # Session tests
go run ./cmd/hangar              # Run locally
```

**Note**: Two tests in `internal/ui/` are currently failing pre-existing issues:
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

## UI Architecture (Bubble Tea)

Hangar uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. Key patterns:

- **State changes** go through `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — never mutate state in `View()`
- **Async work** returns `tea.Cmd` functions that produce `tea.Msg` results dispatched back to `Update()`
- **Dialogs** are structs with `IsVisible()`, `Show()`, `Hide()`, `View()`, `HandleKey()`, `SetSize()` methods
- **Adding a new dialog** requires wiring in 7 places in `home.go`: struct field, init, key routing, mouse guard, trigger key, SetSize, View check

## Session Status Detection

Status is detected via two parallel systems:

1. **Hook-based** (fast path): Claude Code writes JSON to `~/.hangar/hooks/{id}.json` via lifecycle hooks. Detected via `fsnotify`.
2. **Tmux-based** (fallback): Periodic polling of pane title (spinner chars), activity timestamps, and content scanning.

Hook command written to Claude's `settings.json`: `hangar hook-handler`

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
| `W` | Worktree finish (merge + cleanup) |
| `d` | Delete session/group |
| `o` | Open PR in browser |
| `G` | Open lazygit |
| `S` | Settings |
| `/` | Search |
| `?` | Help overlay |
| `Ctrl+Q` | Detach from session |
| `q` | Quit |

## Claude Code Hooks

Hangar installs hooks into Claude Code's `~/.claude/settings.json` to detect session lifecycle events (SessionStart, Stop, UserPromptSubmit, PermissionRequest, etc.). The hook command is `hangar hook-handler`.

To install hooks: `hangar hooks install`
To check status: `hangar hooks status`

## Style / Conventions

- **Theme**: oasis_lagoon_dark (defined in `internal/ui/styles.go`)
- **Error display**: Use `h.setError(err)` for transient status bar messages (auto-dismiss after ~3s)
- **Async patterns**: Background goroutines return typed `tea.Msg` structs — see `sendTextResultMsg`, `worktreeFinishResultMsg` for examples
- **Logging**: `slog` throughout; log files in `~/.hangar/logs/`

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `HANGAR_INSTANCE_ID` | Current session ID |
| `HANGAR_TITLE` | Current session title |
| `HANGAR_TOOL` | Tool in use (claude, shell, etc.) |
| `HANGAR_PROFILE` | Active profile |

## Release

Releases are built via GoReleaser (`go install github.com/goreleaser/goreleaser/v2@latest`):

```bash
goreleaser release --snapshot --clean   # Test release build
goreleaser release                       # Actual release (requires GITHUB_TOKEN)
```

Release artifacts are named `hangar_{version}_{os}_{arch}.tar.gz`.

Homebrew formula is at `mnicholson/homebrew-tap`.
