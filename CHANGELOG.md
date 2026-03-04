# Changelog

All notable changes to Hangar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.1] - 2026-03-04

### Fixed

- **`hangar project add <name> .`** — relative paths (`.`, `../foo`, etc.) are now
  resolved to absolute paths at add-time, so the stored `base_dir` is always correct
  regardless of where `hangar` is later launched from.

## [2.1.0] - 2026-03-03

### Added

- **Tower — Hangar control agent** (`hangar tower`) — a special Claude Code session in
  `~/.hangar/tower/` that acts as a meta-agent for all other sessions. Tower has access
  to a built-in MCP server (`hangar mcp-server`) with 12 tools:
  - Session tools: `hangar_list_sessions`, `hangar_get_session`, `hangar_get_output`,
    `hangar_send_message`, `hangar_start_session`, `hangar_stop_session`,
    `hangar_restart_session`, `hangar_create_session`
  - Todo tools: `hangar_list_todos`, `hangar_create_todo`, `hangar_update_todo`,
    `hangar_delete_todo`
  - The MCP server communicates with the Hangar REST API; the web server is auto-started
    if it is not already running.
  - Tower's working directory, `.mcp.json`, `CLAUDE.md` system prompt, and
    `settings.local.json` (pre-approving `hangar` CLI commands) are all scaffolded
    automatically on first run.
  - **Remote access**: pass `--happy` to wrap Tower with
    [happy](https://github.com/anthropics/happy) (`happy {command}`) for remote browser
    access. Use `--wrapper <cmd>` for any custom wrapper.
  - Run `hangar tower --attach` (or `-a`) to attach the terminal directly; without the
    flag the command starts Tower in the background and exits.

- **`session_type` field** — sessions now carry a `session_type` string (`"tower"` for
  Tower, empty for regular sessions). Exposed in the REST API and persisted in SQLite.

- **Tower badge in TUI** — Tower sessions render a `◈` badge (cyan) in the session list
  instead of the tool name badge, and are always pinned above all project groups.

- **Tower badge in Web UI** — Tower sessions appear at the top of the sidebar with a
  `◈` badge and a thin divider separating them from project groups.

## [2.0.2] - 2026-03-03

### Changed

- **Web UI: Oasis Lagoon Dark / Oasis Dawn theme** — the web UI now uses the same colour
  palette as the TUI. Dark mode uses the full Oasis Lagoon Dark palette (navy `#101825`
  background, `#58B8FD` accent, `#53D390` green, etc.). Light mode uses Oasis Dawn (lagoon
  blue surfaces, dark navy text). All status and PR badge colours now reference theme-aware
  CSS custom properties rather than fixed Tailwind colour names, so they remain legible in
  both modes.

### Fixed

- **Web UI: active/waiting badges now visible in light mode** — status badges previously used
  `text-green-400` / `text-yellow-400` which are too light against pale backgrounds. Badges
  now use high-contrast foreground colours (`#1b491d` green, `#6b2e00` amber) on pastel
  backgrounds in light mode, and the Oasis palette colours on translucent navy surfaces in
  dark mode. PR state and CI check colours in `PRBadge` are also updated.

- **Web UI: delete confirmation mentions worktree side-effect** — for sessions that own a git
  worktree, the delete confirmation prompt now reads "Delete session + worktree?" instead of
  the generic "Delete?" so users understand the filesystem consequence before confirming.

## [2.0.1] - 2026-03-03

### Fixed

- **Web UI: worktree sessions now start on the correct branch** — when creating a session via
  the web UI with "Create in worktree" enabled, the tmux session was starting in the base
  repository directory (on `master`/`main`) instead of the newly created worktree directory.
  The instance was being constructed with the base path before the worktree was created; the
  worktree path was patched onto the instance afterward but the internal tmux `WorkDir` was
  never updated. Fixed by resolving the effective working directory first and passing it to
  the instance constructor, matching the approach used by the CLI `hangar add` command.

- **Web UI: deleting a worktree session now removes the worktree** — deleting a session via
  the web UI was only killing the tmux process and removing the DB record; it did not call
  `git worktree remove` or `git worktree prune`. Added the same `RemoveWorktree` +
  `PruneWorktrees` calls the TUI's `deleteSession` already performs.

## [2.0.0] - 2026-03-03

### Added

- **Embedded Web UI** — a full React + Vite browser dashboard is now compiled into the Go
  binary via `go:embed`. Access it at `http://localhost:47437/ui/` (or over Tailscale when
  `bind_address = "0.0.0.0"`). No separate server process required — starts with the TUI and
  stays running in the background.

  Key features of the web dashboard:
  - **Session list with live status** — running `●`, waiting `◐`, idle `○` updated in
    real time via WebSocket push events
  - **Project sidebar** — all projects shown, including those with no active sessions;
    click to navigate to the project detail page
  - **Session detail** — full xterm.js terminal view with PTY streaming, info bar showing
    project/path/branch/age metadata
  - **Project detail** — base directory, base branch, session count, and an inline
    todo kanban board (todo / in progress / done columns)
  - **Todo board** — add, update status, and delete todos; synced with the TUI's kanban
  - **Light / dark / system theme** — toggle in the top bar; follows OS `prefers-color-scheme`
    in system mode; preference persisted in localStorage
  - **Resizable sidebar** — drag handle for custom sidebar width, also persisted across page
    loads
  - **New session dialog** — create sessions with optional worktree + branch name and
    `--dangerously-skip-permissions` flag

- **`hangar web` subcommand** — lifecycle management for the standalone web server:
  ```
  hangar web start    # run in foreground (background with &); writes ~/.hangar/web.pid
  hangar web stop     # send SIGTERM to the running server via PID file
  hangar web status   # probe /api/v1/status and print URL, uptime, session counts
  ```
  The TUI also starts the API server automatically; `hangar web status` works regardless of
  how the server was started.

- **Sidebar reload on create/delete** — creating or deleting a session via the API now
  immediately triggers a TUI storage reload (bypasses the 2-second poll interval) so the
  sidebar updates without delay.

- **Delete project from web UI** — projects can be removed from the sidebar via a confirm
  button (hover to reveal).

### Changed

- The `/ui/*` routes now return a fully functional web application instead of a 501
  placeholder.

## [1.2.2] - 2026-03-03

### Fixed

- **Shift+Enter now inserts a newline** in the "Send to Session" dialog (`x` key) instead of
  being ignored. The input has been upgraded from a single-line `textinput` to a multi-line
  `textarea`; bare Enter submits, Shift+Enter inserts a newline. The hint text in the dialog
  reflects this.

- **Shift+Enter works in attached sessions** — hangar now sets `extended-keys on` on each
  tmux session at startup. Without this, tmux stripped the Kitty keyboard protocol modifier
  from Shift+Enter before the bytes reached Claude Code, making it indistinguishable from
  plain Enter. Requires tmux 3.2+; silently ignored on older versions.

## [1.2.1] - 2026-03-03

### Fixed

- **PR badge no longer flickers during background refresh** — the sidebar list was using
  the TTL-aware `GetPR` cache accessor, so the `[#NNN]` badge and CI check counts would
  silently disappear for ~1–3 s every 60 s while `gh pr view` re-ran in the background.
  Switched to `HasPREntry` (returns last-known data regardless of TTL) so the badge stays
  visible until the refresh result arrives.

- **CI check counts easier to read** — counts in the session list sidebar were formatted
  as `✕3 ◐2 ✓15` (icon butted against number, parts separated by a single space). Now
  rendered as `✕ 3  ◐ 2  ✓ 15` (space between icon and count, two spaces between groups).

- **Removed redundant PR re-fetch on navigation** — navigating between sessions was
  triggering a `gh pr view` call whenever the 60 s TTL had expired, in addition to the
  background tick that already does this every 2 s. The navigation-triggered fetch has been
  removed; the background tick is the sole refresh path.

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
