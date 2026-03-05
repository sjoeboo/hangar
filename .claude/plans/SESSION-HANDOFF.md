# Session Handoff - 2026-03-05

## What Was Accomplished

- Added CHANGELOG entries for v2.2.0 (auto-open browser) and v2.3.0 (PR management overhaul)
- Designed and implemented daemon-first architecture (PR #40 open)
- TUI no longer hosts embedded API server — probes port, connects via WS, auto-forks daemon if needed
- Added `--detach` / `-d` flag to `hangar web start`
- Added `hook_changed` WS event broadcast from apiserver hooks handler
- Created `DaemonClient` (`internal/ui/daemon_client.go`) with tests
- Removed broken `--web` flag and `runWebInProcess`
- Design doc: `docs/plans/2026-03-04-daemon-first-architecture-design.md`

## Current State

- Branch: `feature/changelog-1` (worktree: `.worktrees/feature-changelog-1`)
- PR #40: https://github.com/sjoeboo/hangar/pull/40 (open, all code committed + pushed)

## ONLY REMAINING TASK: Add v2.4.0 Changelog Entry

Add `## [2.4.0] - 2026-03-05` at the top of CHANGELOG.md (above [2.3.0]).

### Added
- **Daemon-first architecture** — TUI probes port 47437 on startup; if a daemon is running it subscribes via WebSocket for real-time events, otherwise it auto-forks `hangar web start --detach` as a background child (killed when TUI exits). Three clean modes:
  - **TUI only** (`hangar`): daemon auto-started with `--detach`, dies with TUI
  - **Web only** (`hangar web start`): persistent daemon, `hangar web stop` to kill
  - **Both** (`hangar web start` then `hangar`): TUI connects to pre-existing daemon, leaves it running on exit
- **`--detach` / `-d` flag for `hangar web start`** — re-execs the process in the background with stdout/stderr redirected to `~/.hangar/logs/web.log`; returns control immediately. The TUI's auto-fork uses this flag.
- **`hook_changed` WebSocket event** — API server broadcasts `hook_changed` (payload: `instance_id`, `hook_event_name`, `status`) when a Claude Code lifecycle hook fires, enabling TUI clients connected to an external daemon to receive real-time status updates
- **`DaemonClient`** (`internal/ui/daemon_client.go`) — WebSocket client in the TUI translating daemon events into Bubble Tea messages (`hook_changed` → `hookStatusChangedMsg`, `sessions_changed`/`session_created`/`session_deleted` → `storageChangedMsg`), following the same one-shot `tea.Cmd` pattern as `listenForHookChanges`

### Removed
- **`hangar --web` flag** — was broken (port conflict + duplicate `pr.Manager` in same process); TUI auto-start replaces it
- **`runWebInProcess()`** — dead code removed from `cmd/hangar/web_cmd.go`

### Reference link to add at bottom of CHANGELOG:
```
[2.4.0]: https://github.com/sjoeboo/hangar/releases/tag/v2.4.0
```

Then commit + push:
```bash
git add CHANGELOG.md
git commit -m "docs: add v2.4.0 changelog entry for daemon-first architecture"
git push
```

PR #40 is then ready to merge.
