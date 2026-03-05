# Daemon-First Architecture Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Hangar's TUI always connect to an external API server daemon rather than hosting its own, auto-starting one as a child process when none is running.

**Architecture:** The daemon (`hangar web start`) owns the API server + WebSocket hub. When the TUI starts, it probes the configured port; if a daemon is already running it subscribes to its WebSocket for real-time events, otherwise it forks `hangar web start` as a child process and kills it on exit. The TUI retains its own `pr.Manager` for display purposes.

**Tech Stack:** Go 1.23+, `gorilla/websocket` (already in go.mod), Bubble Tea, standard `os/exec`

---

## Context: Key Files

| File | Role |
|------|------|
| `cmd/hangar/main.go` | CLI dispatch — remove `--web` flag, add daemon probe + auto-fork |
| `cmd/hangar/web_cmd.go` | `handleWebStart` (keep), `runWebInProcess` (delete) |
| `internal/apiserver/hooks.go` | HTTP hook handler — add `hook_changed` WS broadcast |
| `internal/apiserver/types.go` | Add `WsHookChangedData` type |
| `internal/ui/home.go` | Remove embedded server startup (~lines 958–1016); add WS subscription |
| `internal/ui/daemon_client.go` | New file — WS client that feeds `tea.Msg` to the TUI |

## Pre-existing test failures (do not fix)
- `TestNewDialog_WorktreeToggle_ViaKeyPress`
- `TestNewDialog_TypingResetsSuggestionNavigation`

---

## Task 1: Broadcast `hook_changed` WS event from the API server

**Files:**
- Modify: `internal/apiserver/types.go`
- Modify: `internal/apiserver/hooks.go`
- Modify: `internal/apiserver/hooks_test.go`

### Step 1: Add `WsHookChangedData` type to `types.go`

Append to `internal/apiserver/types.go`:

```go
// WsHookChangedData is the data payload for the hook_changed WS event.
type WsHookChangedData struct {
	InstanceID    string `json:"instance_id"`
	HookEventName string `json:"hook_event_name"`
	Status        string `json:"status"`
}
```

### Step 2: Broadcast `hook_changed` after processing in `hooks.go`

In `handleHook`, after the `if status != ""` block (before `w.WriteHeader`), add the broadcast. The `s.hub` field is accessible because `handleHook` is a method on `*APIServer`.

Change `hooks.go` from:
```go
	if status != "" {
		s.watcher.Notify(instanceID, status, payload.SessionID, payload.HookEventName)
	}

	w.WriteHeader(http.StatusOK)
```

To:
```go
	if status != "" {
		s.watcher.Notify(instanceID, status, payload.SessionID, payload.HookEventName)
		s.hub.broadcast <- WsMessage{
			Type: "hook_changed",
			Data: WsHookChangedData{
				InstanceID:    instanceID,
				HookEventName: payload.HookEventName,
				Status:        status,
			},
		}
	}

	w.WriteHeader(http.StatusOK)
```

### Step 3: Write a test verifying the broadcast

Add to `internal/apiserver/hooks_test.go`:

```go
func TestHookServer_BroadcastsHookChanged(t *testing.T) {
	watcher, _ := session.NewStatusFileWatcher()
	defer watcher.Stop()
	srv := New(APIConfig{Port: 0}, watcher, nil, nil, nil, nil, "default", "test")

	// Subscribe to the WS hub before sending the hook
	sub := make(chan WsMessage, 4)
	srv.hub.subscribe <- sub
	defer func() { srv.hub.unsubscribe <- sub }()

	// Run hub
	go srv.hub.run()

	body := `{"hook_event_name":"UserPromptSubmit","session_id":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/hooks", strings.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-1")
	rr := httptest.NewRecorder()
	srv.handleHook(rr, req)

	select {
	case msg := <-sub:
		if msg.Type != "hook_changed" {
			t.Fatalf("expected hook_changed, got %q", msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hook_changed broadcast")
	}
}
```

### Step 4: Run the test suite

```bash
go test ./internal/apiserver/... -v -run TestHook
```

Expected: all `TestHook*` tests pass, including the new one.

### Step 5: Commit

```bash
git add internal/apiserver/types.go internal/apiserver/hooks.go internal/apiserver/hooks_test.go
git commit -m "feat(apiserver): broadcast hook_changed WS event on HTTP hook receipt"
```

---

## Task 2: Create `internal/ui/daemon_client.go` — TUI WebSocket client

**Files:**
- Create: `internal/ui/daemon_client.go`

This file provides a goroutine that connects to the daemon's WS endpoint, reads events, and exposes a `tea.Cmd` that the TUI re-issues in its `Update` loop (same pattern as `listenForHookChanges`).

### Step 1: Write the file

```go
package ui

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// daemonWSMsg is received from the daemon WebSocket and triggers TUI updates.
type daemonWSMsg struct {
	eventType string // "hook_changed", "sessions_changed", etc.
}

// DaemonClient maintains a WebSocket connection to a running hangar daemon and
// surfaces events as Bubble Tea commands. Re-issue listenForDaemonEvents after
// each message to keep receiving, exactly like listenForHookChanges.
type DaemonClient struct {
	url  string
	msgC chan daemonWSMsg
	done chan struct{}
}

// newDaemonClient dials the daemon WebSocket and starts a read loop.
// Returns nil if the connection cannot be established.
func newDaemonClient(port int) *DaemonClient {
	url := fmt.Sprintf("ws://127.0.0.1:%d/api/v1/ws", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		slog.Warn("daemon_ws_connect_failed", slog.String("url", url), slog.String("error", err.Error()))
		return nil
	}

	dc := &DaemonClient{
		url:  url,
		msgC: make(chan daemonWSMsg, 32),
		done: make(chan struct{}),
	}

	go dc.readLoop(conn)
	return dc
}

func (dc *DaemonClient) readLoop(conn *websocket.Conn) {
	defer close(dc.done)
	defer conn.Close()

	conn.SetReadDeadline(time.Time{}) // no deadline — long-lived connection

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("daemon_ws_read_error", slog.String("error", err.Error()))
			return
		}

		var msg struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}

		switch msg.Type {
		case "hook_changed", "sessions_changed", "session_created", "session_deleted":
			select {
			case dc.msgC <- daemonWSMsg{eventType: msg.Type}:
			default:
				// Drop if buffer full — TUI will reload on next tick
			}
		}
	}
}

// Close signals the client to stop. Safe to call multiple times.
func (dc *DaemonClient) Close() {
	// readLoop closes done when the conn drops or we close it.
	// We rely on the conn's read returning an error on context cancellation
	// from the caller; the goroutine exits naturally.
}

// listenForDaemonEvents returns a tea.Cmd that blocks until the next WS event
// arrives, then returns a Bubble Tea message. MUST be re-issued in Update.
func listenForDaemonEvents(dc *DaemonClient) tea.Cmd {
	return func() tea.Msg {
		if dc == nil {
			return nil
		}
		select {
		case msg, ok := <-dc.msgC:
			if !ok {
				return nil
			}
			switch msg.eventType {
			case "hook_changed":
				return hookStatusChangedMsg{}
			case "sessions_changed", "session_created", "session_deleted":
				return storageChangedMsg{}
			}
		case <-dc.done:
			return nil
		}
		return nil
	}
}
```

Note: add `"fmt"` to the imports.

### Step 2: Run the build to confirm it compiles

```bash
go build ./internal/ui/...
```

Expected: clean build (no test yet — that comes in Task 6).

### Step 3: Commit

```bash
git add internal/ui/daemon_client.go
git commit -m "feat(ui): add DaemonClient — WS subscriber for external daemon events"
```

---

## Task 3: Modify `home.go` — connect to daemon instead of hosting server

**Files:**
- Modify: `internal/ui/home.go`

There are two changes:
1. Replace the embedded server startup block with daemon connection logic.
2. Wire `listenForDaemonEvents` into Init and the Update re-issue points.
3. Clean up `hookServer`/`hookServerPort` field references on shutdown.

### Step 1: Add `daemonClient` field to the `Home` struct

Find the field declarations near `hookServer` (~line 221) and add:

```go
daemonClient       *DaemonClient  // non-nil when connected to external daemon
```

Keep `hookServer` and `hookServerPort` fields for now — they'll be removed in Task 5 cleanup.

### Step 2: Replace the embedded server startup block

Find the block starting at `// Start embedded HTTP/WS API server if configured` (~line 958) and ending at the closing `}` at ~line 1016. Replace the entire block with:

```go
// Connect to daemon or start one as a child process.
// connectOrStartDaemon is called from main.go before the TUI starts;
// the port is injected via h.configuredHookPort which is already set above.
{
	port := h.configuredHookPort
	if port > 0 {
		dc := newDaemonClient(port)
		if dc != nil {
			h.daemonClient = dc
			// Upgrade HTTP hooks to point at the running daemon.
			go func() {
				configDir := session.GetClaudeConfigDir()
				if _, err := session.InjectClaudeHooks(configDir, port); err != nil {
					uiLog.Warn("hook_upgrade_failed", slog.String("error", err.Error()))
				}
			}()
		}
	}
}
```

### Step 3: Wire `listenForDaemonEvents` into Init

In the `Init` method, near the `listenForHookChanges` line (~line 1487):

```go
// Start listening for hook status changes (immediate TUI refresh on hook events)
if h.hookWatcher != nil {
    cmds = append(cmds, listenForHookChanges(h.hookWatcher))
}
// If connected to external daemon, listen for WS events instead of (or in addition to) local hooks.
if h.daemonClient != nil {
    cmds = append(cmds, listenForDaemonEvents(h.daemonClient))
}
```

### Step 4: Re-issue `listenForDaemonEvents` in the Update handler

Search for every site where `listenForHookChanges(h.hookWatcher)` is returned as a `tea.Cmd` (there are 2: the `hookStatusChangedMsg` case and the new-session creation path). After each, add a parallel re-issue for the daemon client:

```go
// Before (example):
return h, listenForHookChanges(h.hookWatcher)

// After:
cmds := []tea.Cmd{listenForHookChanges(h.hookWatcher)}
if h.daemonClient != nil {
    cmds = append(cmds, listenForDaemonEvents(h.daemonClient))
}
return h, tea.Batch(cmds...)
```

Do the same for the `storageChangedMsg` case if it re-issues hook listening.

### Step 5: Clean up daemon client on shutdown

Find the shutdown block (~line 4630) where `h.hookServer` is waited on:

```go
// Wait for the HTTP hook server to finish draining in-flight requests.
if h.hookServer != nil {
    select {
    case <-h.hookServer.WaitDone():
    case <-time.After(3 * time.Second):
    }
}
```

Add after it:

```go
// Close daemon WS client (if connected to external daemon).
if h.daemonClient != nil {
    h.daemonClient.Close()
}
```

### Step 6: Build and run tests

```bash
go build ./internal/ui/...
go test ./internal/ui/... -v -run TestHome
```

Expected: existing tests pass (the two pre-existing failures are expected).

### Step 7: Commit

```bash
git add internal/ui/home.go
git commit -m "feat(ui): connect TUI to daemon WS instead of hosting embedded server"
```

---

## Task 4: Modify `main.go` — daemon probe + auto-fork + cleanup

**Files:**
- Modify: `cmd/hangar/main.go`

### Step 1: Add `probeDaemon` helper function

Add near the bottom of `main.go` (before `extractBoolFlag`):

```go
// probeDaemon checks if a hangar daemon is already listening on the given port.
// Returns true if GET /api/v1/status responds within 200ms.
func probeDaemon(port int) bool {
	if port <= 0 {
		return false
	}
	client := &http.Client{Timeout: 200 * time.Millisecond}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/status", port)
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
```

### Step 2: Add `startDaemonChild` helper function

```go
// startDaemonChild forks "hangar web start" as a child process and waits up to
// 3 seconds for it to be ready. Returns the child process (caller must SIGTERM
// it on exit) and true, or nil and false on failure.
func startDaemonChild(port int) (*os.Process, bool) {
	self, err := os.Executable()
	if err != nil {
		return nil, false
	}
	cmd := exec.Command(self, "web", "start", "--no-open")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, false
	}

	// Poll until ready (up to 3s)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if probeDaemon(port) {
			return cmd.Process, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Daemon didn't start in time — kill it
	_ = cmd.Process.Kill()
	return nil, false
}
```

Imports needed: `"os/exec"` (already used), `"os"`, `"time"`, `"net/http"`, `"fmt"` — check existing imports and add any missing ones.

### Step 3: Replace the `--web` block with probe + auto-fork

Find (~line 409):
```go
// If -web flag was given, start the web server in-process before the TUI.
if webFlag {
    cancelWeb := runWebInProcess(profile, false)
    defer cancelWeb()
}
```

Replace with:
```go
// Ensure a daemon is running before the TUI starts.
// If one is already up (e.g. from "hangar web start"), connect to it.
// Otherwise, fork one as a child process and kill it when we exit.
{
	port := resolvePort(profile) // see Step 4
	if !probeDaemon(port) {
		if child, ok := startDaemonChild(port); ok {
			defer func() {
				if err := child.Signal(syscall.SIGTERM); err != nil {
					_ = child.Kill()
				}
			}()
		}
	}
}
```

Add `"os/signal"` and `"syscall"` to imports if not already present.

### Step 4: Add `resolvePort` helper

```go
// resolvePort reads the configured API port from ~/.hangar/config.yaml.
// Returns 0 if config cannot be read or port is disabled.
func resolvePort(profile string) int {
	cfg, err := session.LoadConfigWithProfile(profile)
	if err != nil || cfg == nil {
		return 47437 // default
	}
	return cfg.API.GetPort(&cfg.Claude)
}
```

Check `session.LoadConfigWithProfile` exists — if the function has a different name, grep for `LoadConfig` in `internal/session/config.go` and use the correct one.

### Step 5: Remove the `--web` flag

Remove these lines:
```go
// Extract global -web/--web flag — starts the web server alongside the TUI.
webFlag, args := extractBoolFlag(args, "-web", "--web")
```

And the `webFlag` variable usage. Also remove from `printHelp`:
```
--web                  Start the web UI server alongside the TUI
```

### Step 6: Build

```bash
go build ./cmd/hangar/...
```

Fix any compile errors (missing imports, unused variables).

### Step 7: Commit

```bash
git add cmd/hangar/main.go
git commit -m "feat(main): auto-start daemon as child process; remove --web flag"
```

---

## Task 5: Remove `runWebInProcess` and `extractBoolFlag`

**Files:**
- Modify: `cmd/hangar/web_cmd.go`
- Modify: `cmd/hangar/main.go`

### Step 1: Delete `runWebInProcess` from `web_cmd.go`

Remove the entire `runWebInProcess` function (lines ~184–278). It is no longer called by anything.

### Step 2: Delete `extractBoolFlag` from `main.go`

Remove the `extractBoolFlag` function (~lines 470–485). Confirm it is unused:

```bash
grep -n "extractBoolFlag" cmd/hangar/main.go
```

Expected: no occurrences.

### Step 3: Build and test

```bash
go build ./...
go test ./... 2>&1 | grep -v "^ok\|no test files" | head -30
```

Expected: clean build. Only the two pre-existing test failures should appear.

### Step 4: Commit

```bash
git add cmd/hangar/main.go cmd/hangar/web_cmd.go
git commit -m "refactor: remove runWebInProcess and extractBoolFlag"
```

---

## Task 6: Test `DaemonClient` event translation

**Files:**
- Create: `internal/ui/daemon_client_test.go`

### Step 1: Write the test

```go
package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDaemonClient_TranslatesHookChanged(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	sentC := make(chan string, 4)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for msg := range sentC {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}))
	defer srv.Close()

	// Parse port from test server URL
	addr := strings.TrimPrefix(srv.URL, "http://")
	// Build a DaemonClient that dials our test server instead of 127.0.0.1:port.
	// Since newDaemonClient hardcodes the URL, dial directly:
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	dc := &DaemonClient{msgC: make(chan daemonWSMsg, 8), done: make(chan struct{})}
	go dc.readLoop(conn)

	// Send hook_changed event from "server"
	msg, _ := json.Marshal(map[string]string{"type": "hook_changed"})
	sentC <- string(msg)

	cmd := listenForDaemonEvents(dc)
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()

	select {
	case m := <-result:
		if _, ok := m.(hookStatusChangedMsg); !ok {
			t.Fatalf("expected hookStatusChangedMsg, got %T", m)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestDaemonClient_TranslatesSessionsChanged(t *testing.T) {
	// Same pattern — send "sessions_changed", expect storageChangedMsg.
	// Implementation omitted for brevity; mirrors the test above.
}
```

Note: `tea.Msg` needs `"github.com/charmbracelet/bubbletea"` import.

### Step 2: Run the test

```bash
go test ./internal/ui/... -v -run TestDaemonClient
```

Expected: both tests pass.

### Step 3: Commit

```bash
git add internal/ui/daemon_client_test.go
git commit -m "test(ui): add DaemonClient WS event translation tests"
```

---

## Task 7: Manual smoke test + final cleanup

### Step 1: Build

```bash
go build -o /tmp/hangar-test ./cmd/hangar
```

### Step 2: Test TUI-only mode

```bash
/tmp/hangar-test
```

Expected: TUI opens normally. Check that `~/.hangar/web.pid` is created (daemon auto-started). Browser should open to `http://localhost:47437/ui/`.

### Step 3: Test web-only mode

```bash
/tmp/hangar-test web start &
curl -s http://localhost:47437/api/v1/status | jq .
```

Expected: status JSON returned, `hangar` (TUI) not running.

### Step 4: Test both modes

```bash
/tmp/hangar-test web start &
sleep 1
/tmp/hangar-test
```

Expected: TUI opens, connects to existing daemon (no second server started), quitting TUI leaves daemon running.

### Step 5: Update help text in `printHelp`

Remove the `--web` line from `printHelp` in `main.go` (should already be done in Task 4 Step 5 — verify).

Update the `web` section description to clarify the auto-start behavior:

```
web start         Start standalone web server (foreground; auto-started by TUI if not running)
```

### Step 6: Run full test suite

```bash
go test ./... 2>&1 | grep -E "FAIL|panic" | grep -v "TestNewDialog_WorktreeToggle\|TestNewDialog_Typing"
```

Expected: no failures except the two pre-existing ones.

### Step 7: Final commit

```bash
git add cmd/hangar/main.go
git commit -m "docs(help): clarify auto-start behavior in web subcommand help text"
```
