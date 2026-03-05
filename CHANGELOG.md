# Changelog

All notable changes to Hangar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.6.0] - 2026-03-05

### Added

- **TUI: Nav tab bar** — a persistent `Sessions · PRs · Todos` tab row appears below the
  header in all three views. Click tabs or press `[`/`]` to cycle. Active tab uses a
  powerline pill; inactive tabs are plain text.

- **TUI: Labeled filter pills** — Sessions view shows `All · ● Running N · ◐ Waiting N · ○ Idle N`
  below the nav tabs. Click a pill or press `!`/`@`/`#`/`$` to filter; click the active pill
  to clear. Counts update live. Mouse-clickable.

- **TUI: All-todos kanban** — Todos tab shows todos across all projects in one board. Each
  card shows a dim project badge. Press `f` to filter to a specific project.

- **TUI: Project picker for new todos** — pressing `n` in the all-projects todo view opens a
  project picker (pre-selected to the active project) before the form. Pressing `n` when
  already filtered jumps straight to the form.

- **TUI: Todo filter shows all projects** — the `f` project filter now lists all known
  projects from the group tree, not just those with existing todos.

### Fixed

- **TUI: Uniform header bar** — all views use `renderHeaderBar()` with consistent
  `ColorSurface` background. Every rendered element carries an explicit background, eliminating
  terminal-default gaps that made the logo brackets appear as a distinct box.

- **TUI: No background boxes on inactive items** — inactive nav tabs and the `All` filter pill
  no longer carry `ColorSurface` rectangles; they appear as plain text matching other inactive
  items.

- **TUI: No dark line in active filter pill** — pre-rendering the keyboard hint inside the
  pill body caused a mid-body reset making the trailing space dark before the right powerline
  cap. Active pill hints are now plain text.

- **TUI: Filter bar click offset corrected** — filter pill mouse clicks now register on the
  correct row after the separator row was introduced between nav tabs and filter bar.

- **TUI: PR view layout consistency** — PR overview uses `renderHeaderBar()` and includes the
  same separator row between nav tabs and its sub-tabs, matching Sessions view structure.

## [2.5.0] - 2026-03-04

### Added

- **TUI: Session diff view — collapsible file tree** — the `D` key overlay now renders a
  collapsible per-file list identical to the PR Detail overlay's Diff tab. Each row shows the
  file path (colorized green/red/yellow by added/deleted/modified status), a `▶`/`▼` expand
  indicator, and `+N -N` change counts. `j`/`k` navigate between files; `Enter`/`Space`
  expand or collapse hunks; single-file diffs auto-expand. Focus highlight, color palette,
  and key bindings are now identical between the session diff view and the PR diff view.
  Inline `lipgloss.NewStyle()` calls in `renderDiffLine` and `renderHunkHeader` replaced with
  pre-compiled package-level style vars for allocation consistency.

- **TUI: Approve PR from detail overlay** — the `a` key now works in the `PRDetailOverlay`
  (Enter → detail view), not just in the PR list. Hint bar updated to show `a approve`.

- **TUI: PR list sorting** — `S` key cycles through sort columns (`age` → `title` → `author`
  → `state` → `checks`); pressing again on the same column flips direction. Default is
  newest-first by age. Active column shown with `↑`/`↓` arrow in the column header.

- **TUI: Mouse support in PR overview** — scroll wheel moves cursor; single click selects a
  row; clicking an already-selected row opens the detail overlay (equivalent to Enter).

- **WebUI: Click-to-edit todos** — todo titles in the kanban board are now inline-editable:
  click a title to enter edit mode, type to update, press Enter or click away to save.

- **WebUI: PR list sorting** — clicking column headers (Age, Author, Title, Checks, State)
  sorts the list; clicking the same header again reverses direction. `↑`/`↓` indicator shown
  on the active column. Default is newest-first.

### Fixed

- **WebUI: Approve button now consistent across all tabs** — the Approve and Request Changes
  buttons previously only appeared for PRs where `source !== 'mine'`, which caused them to be
  hidden for session-linked PRs also authored by the current user. Buttons now appear for all
  open PRs; GitHub's API prevents actual self-approval at the backend.

- **TUI: PR selection highlight covers entire row** — the old highlight applied a background
  to the pre-ANSI-styled row string, but inner `\x1b[0m` reset sequences cleared the
  background mid-row, causing a patchy look. Each column style now carries the selection
  background individually so the highlight is solid and full-width.

## [2.4.0] - 2026-03-05

### Added

- **Daemon-first architecture** — TUI probes port 47437 on startup; if a daemon is running it
  subscribes via WebSocket for real-time events, otherwise it auto-forks `hangar web start --detach`
  as a background child (killed when TUI exits). Three clean modes:
  - **TUI only** (`hangar`): daemon auto-started with `--detach`, dies with TUI
  - **Web only** (`hangar web start`): persistent daemon, `hangar web stop` to kill
  - **Both** (`hangar web start` then `hangar`): TUI connects to pre-existing daemon, leaves it
    running on exit
- **`--detach` / `-d` flag for `hangar web start`** — re-execs the process in the background with
  stdout/stderr redirected to `~/.hangar/logs/web.log`; returns control immediately. The TUI's
  auto-fork uses this flag.
- **`hook_changed` WebSocket event** — API server broadcasts `hook_changed` (payload:
  `instance_id`, `hook_event_name`, `status`) when a Claude Code lifecycle hook fires, enabling TUI
  clients connected to an external daemon to receive real-time status updates.
- **`DaemonClient`** (`internal/ui/daemon_client.go`) — WebSocket client in the TUI translating
  daemon events into Bubble Tea messages (`hook_changed` → `hookStatusChangedMsg`,
  `sessions_changed`/`session_created`/`session_deleted` → `storageChangedMsg`), following the same
  one-shot `tea.Cmd` pattern as `listenForHookChanges`.

### Removed

- **`hangar --web` flag** — was broken (port conflict + duplicate `pr.Manager` in same process);
  TUI auto-start replaces it.
- **`runWebInProcess()`** — dead code removed from `cmd/hangar/web_cmd.go`.

## [2.3.0] - 2026-03-04

### Added

- **`internal/pr/` package — unified PR data layer** — a new first-class package replaces the
  ad-hoc `prCacheEntry` map in the TUI and the `internalPRCache` in the API server. Key types:
  - `pr.Manager` — lifecycle-managed goroutine pool that fetches and caches PR data; shared by
    the TUI, the API server, and the standalone `hangar web start` server.
  - `pr.PR` — rich PR record with: `Number`, `Title`, `State`, `IsDraft`, `Body`, `URL`,
    `Repo`, `HeadBranch`, `BaseBranch`, `Author`, `ReviewDecision`,
    `ChecksPassed`/`ChecksFailed`/`ChecksPending`, `HasChecks`, `Source`, `SessionID`.
  - `pr.FetchDetail` — fetches full PR detail (files changed, diff hunks, review comments,
    commit messages) for the overlay. Diff content is capped at 512 KB.
  - `pr.RepoFromDir` — exported helper that resolves the GitHub/GHE repo slug from a
    directory's git remote URL.
  - `pr.AddComment`, `pr.ApprovePR`, `pr.RequestChanges` — `gh` CLI wrappers for write
    operations triggered from the overlay.

- **TUI: PR dashboard overhaul** — the `P` key PR overview gains a 4-tab bar:
  - **All** — every PR known to the manager (worktree sessions + Mine + Review Requests)
  - **Mine** — open PRs authored by the current GitHub user (searched via `gh search prs`)
  - **Review Requests** — PRs where the current user has been asked to review
  - **Sessions** — only sessions with an associated open PR (the previous behavior)

  Column layout: `#`, `Age`, `Repo`, `Title`, `Author`, `Checks`, `Review`, `State`. Age is
  formatted compactly (`<1d`, `3d`, `2w`, `14mo`, `2y`).

- **TUI: PR Detail Overlay** (`internal/ui/pr_detail.go`) — pressing `Enter` on any PR in
  the overview opens a full-screen overlay with 5 tabs (cycle with `Tab`):
  - **Overview** — number, state, branch, repo, author, draft status, review decision, CI counts
  - **Description** — PR body rendered with [glamour](https://github.com/charmbracelet/glamour)
    GFM markdown (code blocks, bold, lists, inline code)
  - **Diff** — collapsible file list; each row shows path, status, and `+`/`−` counts. `j`/`k`
    navigate between files; `Enter`/`Space` expand or collapse hunks. Single-file PRs
    auto-expand. The footer hint line updates to show diff-specific key bindings.
  - **Conversation** — review comments and timeline rendered with glamour markdown
  - `c` — add a review comment (reuses the send-text dialog, wired to `pr.AddComment`)
  - `o` — open PR in browser
  - `q` — close overlay
  - All lipgloss styles are pre-compiled at package level to eliminate per-frame allocations.

- **TUI: PR overview quality-of-life**
  - `D` toggle — hide/show draft PRs in the overview table; draft rows are dimmed
  - Review decision icons in the `Review` column: `✓` (approved), `△` (changes requested)
  - `s` key — create a review session for the selected PR: locates the matching local project
    via `RepoFromDir`, creates a worktree session on `pr.HeadBranch`, and opens it
  - Mine / Review Requests tabs refresh automatically after the first session PR is loaded

- **WebUI: PR Overview table** — the `/prs` page switches from a card layout to a
  column table matching the TUI: `#`, `Age`, `Repo`, `Title`, `Author`, `Checks`, `Review`,
  `State`. The `Author` column is hidden on the **Mine** tab (redundant).

- **WebUI: PR Detail drawer** — clicking a PR row opens a side drawer with 4 tabs:
  - **Overview** — state badges, branch, checks, review decision
  - **Description** — PR body rendered via `react-markdown` + `remark-gfm`
  - **Diff** — expandable file hunks; files default to collapsed; capped at 300 lines/file
    with a "show more" button; wrapped in an `ErrorBoundary` so crashes show a scoped error
  - **Conversation** — review comments rendered with `react-markdown`

- **WebUI: Draft toggle + refresh** — a "Hide drafts" button filters draft PRs from the table;
  draft rows are dimmed (`opacity-60`, italic title) when visible. A `⟳` refresh button spins
  while a fetch is in flight. `staleTime` is `0`, `refetchInterval` is 15 s, and
  `refetchOnWindowFocus` is enabled.

- **WebUI: Shift+Enter in terminal** — `xterm.js` collapses `Shift+Enter` to a plain
  carriage return. `TerminalView` now intercepts the key event before xterm handles it and
  sends the Kitty keyboard protocol sequence (`\x1b[13;2u`) directly over the WebSocket, so
  Claude Code can insert a newline instead of submitting.

- **`hangar --web` flag** — a new global flag starts the API + web UI server in-process
  before the TUI launches. Both interfaces share a single process; the browser opens
  automatically once the server is ready. The server shuts down cleanly (including
  `pr.Manager.Stop()`) when the TUI exits.

- **API: new PR endpoints** — five routes added to `internal/apiserver/pr_handlers.go`:

  | Method | Path | Description |
  |--------|------|-------------|
  | `GET` | `/api/v1/prs` | List all PRs (source filter: `all`/`mine`/`review_requested`/`sessions`) |
  | `GET` | `/api/v1/prs/{key}` | Full PR record by `owner/repo#number` key |
  | `GET` | `/api/v1/prs/{key}/detail` | Full detail: diff, conversation, commits |
  | `POST` | `/api/v1/prs/{key}/review/comment` | Add a review comment |
  | `POST` | `/api/v1/prs/{key}/review/state` | Set review state (approve / request-changes) |

- **Standalone web server: PR data without TUI** — `hangar web start` now creates a
  `pr.Manager` and runs a session-polling goroutine (every 90 s) that reads worktree
  sessions from SQLite and calls `UpdateSessionPR` for each, so `/api/v1/prs` returns live
  data even when the TUI is not running.

### Fixed

- **GHE PR detail loading** — `remoteURLToRepo` now preserves the GHE host prefix in the
  repo key so `FetchDetail` can resolve the correct API endpoint for GitHub Enterprise repos.
- **Mine / Review Requests tabs** — removed fields not supported by `gh search prs` that
  caused empty results on some GitHub API versions.
- **Draft state detection** — `fetchPRInfo` now explicitly requests `isDraft` in the `gh pr
  view` JSON template; previously draft PRs showed as non-draft in the overlay.
- **Author field blank in All / Sessions tabs** — `FetchSessionPR` now includes `author`
  in the field list passed to `gh pr view`.
- **`os.Environ()` mutation hazard** — four sites in `fetch.go` that appended to
  `os.Environ()` now use a defensive copy to avoid potential shared-backing-array corruption
  across concurrent fetch goroutines.
- **WebUI: sidebar PR badge counts all PRs** — the count badge in the nav sidebar now
  reflects the total from `pr.Manager` (all tabs) rather than only session-owned PRs.
- **WebUI: null review comments** — Go's nil-slice serialization was producing `null` for
  `review.comments`; the client now guards against this before iterating.

## [2.2.0] - 2026-03-04

### Added

- **Auto-open browser on `hangar web start`** — when the web server starts successfully,
  Hangar automatically opens the UI URL in the default browser (`open` on macOS,
  `xdg-open` on Linux, `rundll32` on Windows). Pass `--no-open` to suppress this behavior.
  The URL is always printed to stdout so you can open it manually regardless.

- **Readiness poll before browser open** — Hangar polls `/api/v1/status` in a goroutine
  until the server responds before opening the browser tab, so the tab never hits
  "connection refused" while the server is still binding.

- **Browser opens for already-running server** — if `hangar web start` detects the server
  is already running, it now opens the browser (respecting `--no-open`) rather than exiting
  silently.

## [2.1.2] - 2026-03-04

### Added

- **Web UI: PR Overview page** (`/prs`) — a dedicated view listing all sessions that have a
  pull request, sorted by state (OPEN → DRAFT → MERGED → CLOSED). Accessible via the new
  `⎇ PRs` nav link in the sidebar, which shows a live count badge of open/draft PRs.

- **Web UI: PR status auto-refresh** — the API server now runs a background PR refresh loop
  that calls `gh pr view` for every worktree session every 60 seconds and broadcasts a
  `sessions_changed` event so connected browsers pick up fresh CI check counts without manual
  reload. This also fixes PR data being entirely absent when running `hangar web start` in
  standalone mode (previously `getPRInfo` was `nil` in that code path).

### Fixed

- **Web UI: PR check counts show all statuses simultaneously** — the PR badge previously
  used a ternary that displayed only the "worst" status (failures, then pending, then a bare
  checkmark). It now renders all three independently — `✗N` (red failures), `●N` (yellow
  pending), `✓N` (green passed) — matching the TUI's behaviour. The fix applies everywhere
  the badge is used: session list sidebar, PR overview, and session detail header.

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

[2.5.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.5.0
[2.4.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.4.0
[2.3.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.3.0
[2.2.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.2.0
[1.0.0]: https://github.com/sjoeboo/hangar/releases/tag/v1.0.0
