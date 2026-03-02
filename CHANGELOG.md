# Changelog

All notable changes to Hangar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.2] - 2026-03-02

### Fixed

- **Review session: ghost row during worktree creation** — pressing Enter on the confirm step now
  shows a "Creating worktree" spinner row in the session list while the async fetch + worktree
  creation runs, matching the behaviour of the regular new-session worktree flow.
- **Review session: session starts as shell instead of claude** — the review session was created
  with `inst.Command` unset (`""`), causing `buildClaudeCommandWithMessage` to fall through to an
  empty command and open a bare shell instead of starting Claude. The command is now set
  explicitly to `"claude"` before `Start()` is called.
- **Review session: auto-sent `/pr-review` command removed** — the dialog no longer automatically
  sends `/pr-review <pr>` after session creation. Not every user has this slash command; the
  session now starts at a plain Claude prompt so users can run whatever review workflow they prefer.

## [1.1.1] - 2026-02-28

### Fixed

Potential orphaned hook configs

## [1.1.0] - 2026-02-28

### Added

- **HTTP hooks (Claude Code ≥ 2.1.63)** — hangar now embeds a lightweight HTTP server
  (`127.0.0.1:2437` by default) that receives Claude Code lifecycle events via `POST /hooks`
  instead of spawning a `hangar hook-handler` subprocess for every event. This eliminates
  per-event subprocess overhead and enables near-instant TUI status updates.
- **Instant TUI status refresh** — hook-driven status changes (running → waiting → idle)
  now appear in ~100 ms via a `hookChangedCh` channel wired into the Bubble Tea event loop,
  replacing the previous up-to-4-second polling delay.
- **Version-aware hook injection** — `hangar hooks install` auto-detects the installed
  Claude Code version and injects `type: "http"` hooks for ≥ 2.1.63, falling back to the
  existing `type: "command"` hook for older versions. On TUI startup, command hooks are
  silently upgraded to HTTP hooks when the version threshold is newly met.
- **`hangar hooks status` hook type reporting** — the status command now shows
  `Type: HTTP (Claude >= 2.1.63)` or `Type: command (hangar hook-handler)` to make the
  active hook mechanism visible.
- **`hookServerPort` config option** — set `hookServerPort: 0` in `~/.hangar/config.yaml`
  to disable HTTP hooks and use the command hook fallback only.

### Fixed

- **Orphaned HTTP hook entries** — hangar now detects and removes `{"type":"http"}` hook
  entries with no `url` that were written to `~/.claude/settings.json` by a buggy intermediate
  version. These stale entries caused Claude Code to reject `settings.json` entirely on startup
  with `url: Expected string, but received undefined`. The cleanup runs automatically on the
  next `hangar hooks install` or TUI startup.

- **Create New Project dialog** — rewrote `View()` to eliminate the dark rectangle
  artifact that appeared next to labels; input fields now resize dynamically with the
  terminal; added inline validation errors and keyboard hints to all four dialog modes
  (Create, Rename, Move, RenameSession).

### Changed

- Added `golangci-lint` configuration (`.golangci.yml`) to the repository.

## [1.0.4] - 2026-02-27

### Changed

- **Bottom help bar** — trimmed session hints to reduce noise; removed `c`, `p`,
  `t`, `G` from primary hints (all still accessible via `?`); compact bar also
  drops `R` and `f` for narrower terminals
- **`v` key in help bar** — "Review" hint now appears alongside `W` ("Finish")
  whenever a worktree session is selected, making the PR review workflow
  discoverable without opening `?`

### Added

- **README key bindings** — reorganised into three grouped tables (Sessions,
  Worktrees & PRs, Navigation & Projects); added previously undocumented keys:
  `v` (review PR), `R` (restart), `f`/`F` (fork), `c` (copy), `K`/`J`
  (reorder), `1`–`9` (jump to project), `P` (PR overview)

## [1.0.3] - 2026-02-27

Fix up CI/release tooling

## [1.0.0] - 2026-02-24

Initial release of **Hangar** — a focused fork of
[agent-deck](https://ghe.spotify.net/mnicholson/agent-deck) by @mnicholson, trimmed and
re-tuned for solo Claude Code workflows.

### Forked from agent-deck

Hangar started as a personal fork of agent-deck v0.19.14. The upstream project
is a full-featured multi-agent terminal manager with support for Claude, Gemini,
Codex, OpenCode, conductor orchestration, Slack/Telegram bridges, a web UI, and
an MCP pool. Hangar strips all of that back to what one person needs: a clean
TUI for managing Claude Code sessions.

Thanks to @mnicholson and all upstream contributors for the excellent foundation.

### Removed (scope reduction from upstream)

- **Conductor system** — multi-conductor orchestration, heartbeat daemons,
  Telegram/Slack bridge, transition notifier daemon, and all conductor CLI
  subcommands
- **Web UI** — `hangar web` server, WebSocket streaming, push notifications, PWA
- **Non-Claude agent support** — Gemini, OpenCode, Codex, and generic shell
  sessions (tool list reduced to `claude` + `shell`)
- **MCP pool** — global shared MCP proxy pool, pool health monitor, pool quit
  dialog, and all `mcp-proxy` infrastructure
- **Skills management** — skill attach/detach, skill sources, Skills Manager TUI
  dialog (`s` key), and `hangar skill` CLI subcommands
- **Subgroups** — nested group hierarchy replaced with flat project list

### Added

- **Projects system** — flat project list replaces groups/subgroups; `p` key
  opens a new project dialog; `1`–`9` jump to projects; sessions display an
  `in project:` label in the new-session wizard
- **Oasis Lagoon Dark status bar** — custom powerline-style theme with rounded
  window pills and color-coded session status
- **ANSI color preview** — live terminal preview preserves ANSI color sequences;
  two-pointer plain/rendered rendering prevents MaxWidth from stripping OSC codes
- **PR status in worktree preview** — fetches `gh pr view` (number, title, state,
  CI check rollup) with a 60-second TTL cache; displayed in the worktree section
  of the preview pane
- **`o` key — open PR in browser** — `exec.Command("open", url)` launches the
  PR URL from the cache; hint appears in help bar when a PR is detected
- **Send-text dialog** — `x` key opens a modal to type and send a message
  directly to the selected session without attaching
- **`G` key → lazygit** — opens lazygit in a new tmux window inside the
  session's tmux session, pointed at the working directory
- **Minimal notifications default** — status bar shows `⚡ N │ ◐ N` icon+count
  summary by default (was opt-in); counts all sessions including current
- **`capture-pane -e`** — preserve escape sequences in tmux capture for accurate
  ANSI preview rendering
- **Worktree base branch auto-update** — `UpdateBaseBranch()` refreshes the
  stored base branch when the default remote branch changes
- **Source-build installer** (`install.sh`) — validates Go and tmux, builds from
  source with version embedding, installs to `~/.local/bin`, configures tmux
  (mouse, clipboard, 256-color, 50k history), and optionally installs Claude
  Code hooks; supports `--non-interactive`, `--skip-tmux-config`, `--skip-hooks`,
  `--dir`
- **`CLAUDE.md`** — project documentation for Claude Code sessions

### Changed

- **Identity**: binary and config directory renamed from `agent-deck` /
  `~/.agent-deck` to `hangar` / `~/.hangar`; tmux session prefix `agent_deck_`
  → `hangar_`; hook command `"agent-deck hook-handler"` → `"hangar hook-handler"`
- **`AGENT_DECK_SESSION_ID` removed** — replaced entirely by `HANGAR_INSTANCE_ID`
- **`G` key** — was jump-to-bottom; now opens lazygit. `gg` = top, `G` = lazygit
- **Help overlay** — GROUPS section renamed to PROJECTS; updated all key
  descriptions to reflect removed features and new bindings
- **`skills/agent-deck/`** → **`skills/hangar/`** — renamed and updated

### Fixed

- **Crash on new session** — nil pointer dereference in `renderHelpBarFull`,
  `renderHelpBarCompact`, and the `o` key handler when `prCache` contains a nil
  entry (sentinel for "no PR found"). Fixed with `pr != nil` guard at all three
  sites
- **New session dialog** — `Enter` during project-picker step was intercepted by
  `home.go` before the dialog could handle it; fixed with `IsChoosingProject()`
  guard
- **Status bar** — fixed pill rendering, window tab rounding, and project label
  display in session list
- **Help text** — removed stale MCP/Skills references; corrected key descriptions

[1.0.0]: https://github.com/sjoeboo/hangar/releases/tag/v1.0.0
