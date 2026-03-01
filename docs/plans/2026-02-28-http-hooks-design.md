# Design: HTTP Hooks + Real-Time TUI Status Updates

**Date:** 2026-02-28
**Branch:** json-http-hooks
**Status:** Approved

## Problem

Two related issues motivate this work:

1. **Claude Code 2.1.63+ supports HTTP hooks** (`type: "http"`): instead of spawning a subprocess for every hook event, Claude POSTs JSON to a local endpoint. This is cleaner, faster, and avoids subprocess overhead per hook event.

2. **TUI status updates are not reactive.** Hook-based status changes can take up to ~4 seconds to appear:
   - `StatusFileWatcher.onChange` is `nil` — no proactive TUI wake-up on file change
   - `handleTick` only calls `triggerStatusUpdate()` if the user was active within the last 2 seconds — passive watchers get no updates at all

## Goals

- Support HTTP hooks as the preferred delivery mechanism for Claude ≥ 2.1.63
- Fall back to the existing command hook (`hangar hook-handler`) for older versions
- Status changes appear in the TUI near-instantly (~100–200ms) regardless of user activity
- Status files are still written (for startup catchup on TUI relaunch)
- No persistent background daemon required; HTTP server lifecycle is tied to the TUI

## Non-Goals

- Supporting hook delivery when the TUI is not running (HTTP fails silently; command hook fallback writes files when it runs, but HTTP doesn't without the server)
- Persistent hook server daemon (separate process)

## Architecture

```
Claude fires hook event
        │
        ├─── [cmd hook, Claude < 2.1.63] ──→ hangar hook-handler subprocess
        │                                           │
        ├─── [http hook, Claude ≥ 2.1.63] ──→ hangar HTTP server (localhost:PORT/hooks)
        │                                           │
        └───────────────────────────────────────────▼
                                         status map update (in-memory)
                                         status file write (~/.hangar/hooks/{id}.json)
                                         hookChangedCh ← non-blocking send
                                                 │
                                        listenForHookChanges() tea.Cmd
                                                 │
                                        hookStatusChangedMsg → Update() → View()
                                        (instant re-render, no tick wait)
```

## Components

### 1. Reactive hook notification channel (TUI latency fix)

Add a `hookChangedCh chan struct{}` (buffered, size 1) to `StatusFileWatcher`.

`processFile()` sends to it non-blockingly after updating the `statuses` map:
```go
select {
case w.hookChangedCh <- struct{}{}:
default: // already a pending notification, skip
}
```

Add `NotifyChannel() <-chan struct{}` method to expose the channel.

Remove the now-unused `onChange func()` field (it was always `nil`).

**TUI wiring in `home.go`:**

- Add `listenForHookChanges(watcher *session.StatusFileWatcher) tea.Cmd` — blocks on `NotifyChannel()`, returns `hookStatusChangedMsg{}`
- Arm it during `Init()` alongside `listenForReloads`
- `case hookStatusChangedMsg:` in `Update()`:
  - Apply latest hook statuses to visible instances immediately
  - Invalidate `cachedStatusCounts`
  - Re-arm: `return h, listenForHookChanges(h.hookWatcher)`

This makes status changes appear as fast as fsnotify fires (~100ms debounce), fully independent of the user-activity gate.

### 2. Embedded HTTP hook server (`internal/hookserver/`)

New package with a single exported type: `HookServer`.

```go
type HookServer struct {
    port     int
    statusCh chan<- hookEvent  // sends to hook watcher
    server   *http.Server
}
```

**Endpoint:** `POST /hooks`

- Reads `X-Hangar-Instance-Id` header (the `HANGAR_INSTANCE_ID` from tmux env)
- Reads JSON body — same `hookPayload` struct as `hook_handler.go`
- Applies the same `mapEventToStatus()` logic
- Writes status file (same atomic write as `writeHookStatus`)
- Notifies the hook watcher's channel for instant TUI refresh
- Always returns `200 OK` — never blocks Claude

**Lifecycle:**
- Started in `Home.Init()` alongside the hook watcher (only when hooks are enabled)
- Stopped in the shutdown path via `context.Context` cancellation
- Port: configurable via hangar user config (`hookServerPort`, default `2437`)
- Bind address: `127.0.0.1` only (not 0.0.0.0)

**Port:** `2437` is the default — mnemonic: **H**ang**A**r (no good fit, just pick a reasonable unprivileged port). Stored in `UserConfig.HookServerPort int` with a `GetHookServerPort()` accessor defaulting to 2437.

### 3. Version-aware hook injection (`session/claude_hooks.go`)

New function: `DetectClaudeVersion() (semver, error)` — runs `claude --version`, parses the output.

Modify `InjectClaudeHooks(configDir string)` to accept a port parameter (or read from config):

```go
func InjectClaudeHooks(configDir string, port int) (bool, error)
```

**If Claude ≥ 2.1.63:** inject `type: "http"` hook entry:
```json
{
  "type": "http",
  "url": "http://127.0.0.1:2437/hooks",
  "headers": {
    "X-Hangar-Instance-Id": "$HANGAR_INSTANCE_ID"
  },
  "allowedEnvVars": ["HANGAR_INSTANCE_ID"],
  "timeout": 5
}
```

**If Claude < 2.1.63 or version undetectable:** inject existing `type: "command"` entry (unchanged behavior).

**On TUI startup:** always re-check the installed Claude version and silently upgrade from `command` → `http` if the version threshold is newly met. This handles users who update Claude while hangar is installed.

The check is: read `settings.json`, detect current hook type for each event, compare to what _should_ be there given the detected version, and re-inject if they differ.

### 4. Removal / status detection updates

`RemoveClaudeHooks` and `CheckClaudeHooksInstalled` need to handle both `type: "command"` (matching `hangar hook-handler`) and `type: "http"` (matching the hangar URL pattern `127.0.0.1:*/hooks`).

`hangar hooks status` should report which hook type is active.

## File Changes

| File | Change |
|------|--------|
| `internal/session/hook_watcher.go` | Add `hookChangedCh`, `NotifyChannel()`, remove `onChange` |
| `internal/session/claude_hooks.go` | Add `DetectClaudeVersion()`, update `InjectClaudeHooks()` for http type, update remove/check for both types |
| `internal/hookserver/server.go` | **New** — HTTP hook server |
| `internal/ui/home.go` | Add `hookChangedCh` listener, start HTTP server, remove `onChange: nil` |
| `internal/ui/update_handlers.go` | Add `hookStatusChangedMsg` handler |
| `internal/session/config.go` | Add `HookServerPort` to `UserConfig` + accessor |
| `cmd/hangar/hook_handler.go` | Minor: share `mapEventToStatus` logic with hookserver |

## Testing

- Unit tests for `DetectClaudeVersion()` — version string parsing edge cases
- Unit tests for HTTP hook server — valid/invalid payloads, missing header, wrong method
- Unit tests for `InjectClaudeHooks` — both command and http injection, upgrade path
- Integration test for `listenForHookChanges` — channel fires → `hookStatusChangedMsg` delivered

## Rollback / Compatibility

- If HTTP server fails to start (port in use), log a warning and fall back to command hook injection
- Command hook path is fully preserved; nothing is removed
- `hangar hooks install` continues to work as before; it now also starts the HTTP server path if version supports it
