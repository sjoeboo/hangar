# Design: Daemon-First Architecture

**Date:** 2026-03-04
**Status:** Approved

## Problem

Hangar currently has three separate code paths for the API server + `pr.Manager`, all duplicating the same logic:

| Mode | Server owner | `pr.Manager` owner |
|------|-------------|-------------------|
| `hangar` (TUI) | TUI (`home.go:1000`) | TUI (`home.go:775`) |
| `hangar web start` | `handleWebStart()` | `handleWebStart()` |
| `hangar --web` | Race: `runWebInProcess()` wins port, TUI fails silently | Both — two independent pollers |

`hangar --web` is actively broken: both `runWebInProcess()` and the TUI attempt to bind port 47437 in the same process. One silently fails, leaving HTTP hooks disabled and two `pr.Manager` instances running. PR data is not shared between them.

## Goal

Users should be able to run Hangar in any of three modes without manual coordination:

- **TUI only** — `hangar`: terminal interface only
- **Web only** — `hangar web start`: browser dashboard only
- **Both** — `hangar web start &` then `hangar`: both interfaces, sharing one server

PR data and session status must refresh correctly in all three modes, with no dependency between the TUI and web UI.

## Design

### Three Modes

| Mode | How to start | Daemon lifecycle |
|------|-------------|-----------------|
| TUI only | `hangar` | Auto-started as child process; dies when TUI exits |
| Web only | `hangar web start` | Persistent; killed via `hangar web stop` or SIGTERM |
| Both | `hangar web start` then `hangar` | Pre-existing daemon; TUI leaves it running on exit |

**Lifecycle rule:** the daemon's lifetime matches how it was started, not who connects to it.

### TUI Startup Sequence

```
hangar starts
  → probe GET /api/v1/status (200ms timeout)
  │
  ├─ [daemon running]
  │    skip embedded server startup
  │    subscribe to daemon WebSocket for hook/status events
  │    record daemon as "pre-existing" (do not kill on TUI exit)
  │
  └─ [no daemon]
       fork `hangar web start` as child process
       poll /api/v1/status until ready (≤3s, 100ms interval)
       subscribe to WebSocket
       record daemon as "TUI-owned" (send SIGTERM on TUI exit)
```

### WebSocket Hook Events

The TUI currently receives real-time status changes via `hookChangedCh` — an in-process Go channel. When connecting to an external daemon, this channel is unavailable.

The daemon will broadcast a new `hook_changed` WebSocket event carrying the same payload the HTTP hook endpoint currently processes. The TUI's WebSocket client translates `hook_changed` events back into the existing `hookChangedMsg` so the rest of `home.go` is unaffected.

The daemon already broadcasts `sessions_changed`; `hook_changed` is an additive event type on the same connection.

### `pr.Manager` Ownership

Each process maintains its own `pr.Manager`:
- The daemon owns one (used by the API server and web UI)
- The TUI owns one (used for TUI display: PR overview, preview pane, detail overlay)

They run independently in separate processes and do not conflict. Unifying them into a single source of truth (TUI reads PR data from the daemon's API) is explicitly deferred — it requires replacing all `h.prManager.*` call sites in `home.go` with API calls and is a larger refactor.

## What Changes

### Removed
- `hangar --web` global flag (`extractBoolFlag`, `printHelp` entry)
- `runWebInProcess()` function in `web_cmd.go`
- Embedded server startup block in `home.go` (~lines 970-1005)
- `h.hookServer *apiserver.APIServer` field on the TUI model
- `h.hookServerPort int` field on the TUI model

### Added
- **Daemon probe** in `main.go` before TUI init: `GET /api/v1/status` with 200ms timeout
- **Auto-start logic** in `main.go`: fork `hangar web start`, track child PID, SIGTERM on exit
- **WS client** in `home.go`: connects to `ws://localhost:{port}/api/v1/ws` on startup
- **`hook_changed` event** in `internal/apiserver/server.go`: broadcast when hook POST is received
- **WS->msg translation** in `home.go`: `hook_changed` -> `hookChangedMsg`; `sessions_changed` -> `storageChangedMsg`

### Kept Unchanged
- `hangar web start / stop / status` — explicit persistent daemon control
- `h.prManager` in the TUI — still owns its own polling
- HTTP hook injection logic — hooks still POST to port 47437; daemon always owns that port
- All `internal/apiserver/` routes and WebSocket command handling

## File Impact Summary

| File | Change |
|------|--------|
| `cmd/hangar/main.go` | Remove `--web` flag; add daemon probe + auto-start |
| `cmd/hangar/web_cmd.go` | Remove `runWebInProcess()` |
| `internal/ui/home.go` | Remove embedded server startup; add WS client; add `hook_changed` handler |
| `internal/apiserver/server.go` | Add `hook_changed` broadcast in POST `/hooks` handler |
| `internal/apiserver/types.go` | Add `hook_changed` event type constant |

## Out of Scope

- Unifying `pr.Manager` into a single daemon-owned instance (future work)
- `hangar web start` as a login item / launchd / systemd service
- Multi-user or authenticated daemon access
