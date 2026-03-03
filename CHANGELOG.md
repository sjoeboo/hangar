# Changelog

All notable changes to Hangar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-03-03

### Added

- **Full REST + WebSocket API** — the minimal `POST /hooks` webhook receiver has been replaced
  with a complete HTTP API server (`internal/apiserver`). All endpoints share the same port
  (default `47437`) alongside the backward-compatible `/hooks` route. No authentication —
  designed for Tailscale network trust.

  | Method | Path | Description |
  |--------|------|-------------|
  | `GET` | `/api/v1/status` | Server uptime, version, session counts by status |
  | `GET` | `/api/v1/sessions` | List all sessions with live status |
  | `POST` | `/api/v1/sessions` | Create and start a new session |
  | `GET/PATCH/DELETE` | `/api/v1/sessions/{id}` | Get, update title/group/parent, or delete |
  | `POST` | `/api/v1/sessions/{id}/start` | Start session (optional initial message) |
  | `POST` | `/api/v1/sessions/{id}/stop` | Kill tmux session |
  | `POST` | `/api/v1/sessions/{id}/restart` | Restart session |
  | `POST` | `/api/v1/sessions/{id}/send` | Send a message to a running session |
  | `GET` | `/api/v1/sessions/{id}/output` | Get latest captured prompt/output |
  | `GET/POST` | `/api/v1/projects` | List or create projects |
  | `GET/PATCH/DELETE` | `/api/v1/projects/{id}` | Get, update, or delete a project |
  | `GET/POST` | `/api/v1/todos` | List (by project path) or create todos |
  | `GET/PATCH/DELETE` | `/api/v1/todos/{id}` | Get, update, or delete a todo |
  | `GET` | `/api/v1/ws` | WebSocket — real-time push events + command channel |
  | `GET` | `/ui/*` | 501 placeholder for future embedded web UI |

- **WebSocket real-time events** — connected clients receive `session_created`,
  `session_updated`, `session_deleted`, and `sessions_changed` push events whenever
  hook status changes or storage is modified. Clients can also send `send_message`,
  `stop_session`, and `ping` commands over the same connection.

- **PR info in session API responses** — `SessionResponse` now includes a `pr` object with
  PR number, title, state, URL, and CI check counts (`checks_passed`, `checks_failed`,
  `checks_pending`) sourced from the TUI's live PR cache.

- **Startup eager PR fetch** — the TUI now fires `gh pr view` for all worktree sessions with
  no cached PR data on initial load, so the API serves complete PR info immediately rather
  than waiting for a manual `P` key press or the first background poll tick.

- **Configurable bind address** — new `[api]` section in `~/.hangar/config.yaml`:
  ```yaml
  api:
    port: 47437         # default; backward-compat: also reads [claude] hook_server_port
    bind_address: "0.0.0.0"  # default; was hardcoded to 127.0.0.1
  ```
  Setting `bind_address: "0.0.0.0"` (the new default) makes the API reachable over Tailscale.

- **CORS middleware** — all API endpoints include permissive CORS headers
  (`Access-Control-Allow-Origin: *`), appropriate for the Tailscale trust model, enabling
  web frontends and mobile apps to call the API without proxy configuration.

## [1.1.6] - 2026-03-02

### Added

- **CI check counts in the session sidebar** — worktree sessions with an open PR now show
  inline CI status badges (`✕N` failed, `◐N` pending, `✓N` passed) directly in the session
  list alongside the PR number badge. Counts are coloured red/yellow/green in normal mode and
  match the selection highlight when the row is focused, keeping the list readable at a glance.

## [1.1.5] - 2026-03-02

### Fixed

- **`hangar hooks install` now installs HTTP hooks** — the CLI command was hardcoding port `0`,
  which caused it to always install command-type hooks regardless of the configured
  `hookServerPort`. It now reads the port from `~/.hangar/config.yaml` (defaulting to `47437`),
  matching the behaviour of the TUI's auto-install path. Users who ran `hooks uninstall` then
  `hooks install` and got command hooks instead of HTTP hooks should re-run `hangar hooks install`
  after updating.

## [1.1.4] - 2026-03-02

### Changed

- **HTTP hook server default port raised to 47437** — the previous default (`2437`) sat in
  the low registered-port range and could conflict with other local services. Port `47437`
  is well below the macOS ephemeral range (49152+) while being high enough to avoid common
  developer tooling. Existing installs that relied on the old default will need to
  re-run `hangar hooks install` to update the injected hook URL, or set
  `hookServerPort: 47437` explicitly in `~/.hangar/config.yaml`.

## [1.1.3] - 2026-03-02

### Fixed

- **Ctrl+R now refreshes PR status for all worktree sessions** — previously only the currently
  selected session had its PR cache invalidated and re-fetched. Now every worktree session gets
  a fresh `gh pr view` call so the PR overview (`P`) and preview pane show up-to-date data.
- **Background PR refresh covers all sessions** — the periodic tick-based PR poller previously
  only checked the selected session against the 60-second TTL. It now iterates all worktree
  sessions, keeping every PR status current in the background without manual intervention.

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
