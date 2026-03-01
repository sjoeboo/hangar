# HTTP Hooks + Real-Time TUI Status Updates — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the subprocess-per-event command hook with an embedded HTTP server receiving Claude Code webhook POSTs, and fix the TUI status refresh so changes appear instantly rather than with up to 4s lag.

**Architecture:** Embed a lightweight HTTP server (`net/http`, `127.0.0.1:2437`) in the TUI process. Inject `type: "http"` hooks into Claude's settings when Claude ≥ 2.1.63; fall back to existing `type: "command"` hooks otherwise. A new `hookChangedCh chan struct{}` in `StatusFileWatcher` replaces the always-nil `onChange` callback and wires into Bubble Tea's event loop for instant re-renders.

**Tech Stack:** Go standard library (`net/http`, `os/exec`, `regexp`), existing `fsnotify`, Bubble Tea `tea.Cmd` channel pattern (mirrors `StorageWatcher`).

---

### Task 1: Add reactive notification channel to StatusFileWatcher

**Files:**
- Modify: `internal/session/hook_watcher.go`
- Modify: `internal/session/hook_watcher_test.go`

The always-nil `onChange func()` field causes hook file writes to update the in-memory map silently with no TUI wake-up. Replace it with a `chan struct{}` that the TUI can block on via a `tea.Cmd`, exactly like `StorageWatcher.reloadCh`.

**Step 1: Write the failing test**

Add to `internal/session/hook_watcher_test.go`:

```go
func TestStatusFileWatcher_NotifiesOnProcessFile(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)

	w := &StatusFileWatcher{
		hooksDir:      hooksDir,
		statuses:      make(map[string]*HookStatus),
		hookChangedCh: make(chan struct{}, 1),
	}

	filePath := filepath.Join(hooksDir, "inst-notify.json")
	data, _ := json.Marshal(map[string]any{
		"status": "running", "session_id": "s1", "event": "UserPromptSubmit", "ts": time.Now().Unix(),
	})
	_ = os.WriteFile(filePath, data, 0644)

	w.processFile(filePath)

	select {
	case <-w.NotifyChannel():
		// success — notification sent
	default:
		t.Fatal("Expected notification on hookChangedCh after processFile")
	}
}

func TestStatusFileWatcher_NotifyChannelNotBlockedOnSecondFire(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)

	w := &StatusFileWatcher{
		hooksDir:      hooksDir,
		statuses:      make(map[string]*HookStatus),
		hookChangedCh: make(chan struct{}, 1),
	}

	filePath := filepath.Join(hooksDir, "inst-x.json")
	data, _ := json.Marshal(map[string]any{
		"status": "running", "session_id": "s1", "event": "UserPromptSubmit", "ts": time.Now().Unix(),
	})
	_ = os.WriteFile(filePath, data, 0644)

	// Fire twice without draining — second fire must not block
	w.processFile(filePath)
	w.processFile(filePath) // must not block even though channel is full
	// Reaching here without deadlock = pass
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/session/... -run TestStatusFileWatcher_Notify -v
```
Expected: compile error — `hookChangedCh` field doesn't exist yet, `NotifyChannel` method doesn't exist.

**Step 3: Add `hookChangedCh` field and `NotifyChannel` method**

In `internal/session/hook_watcher.go`:

Replace the `onChange func()` field in `StatusFileWatcher`:
```go
// Before (remove this line):
onChange func()

// After (add this line):
hookChangedCh chan struct{} // non-blocking notification channel; buffered 1
```

Update `NewStatusFileWatcher` — change signature from `onChange func()` to remove that parameter entirely, and initialize the channel:
```go
func NewStatusFileWatcher(onChange func()) (*StatusFileWatcher, error) {
```
becomes:
```go
func NewStatusFileWatcher() (*StatusFileWatcher, error) {
```

Inside, add channel initialization:
```go
hookChangedCh: make(chan struct{}, 1),
```

Remove the `onChange: onChange` line from the return struct.

Add `NotifyChannel` method:
```go
// NotifyChannel returns a receive-only channel that receives a signal whenever
// a hook status file is processed. Buffered 1 — drain it to re-arm.
func (w *StatusFileWatcher) NotifyChannel() <-chan struct{} {
	return w.hookChangedCh
}
```

In `processFile`, replace the `onChange` call block at the bottom:
```go
// Before (remove):
if w.onChange != nil {
    w.onChange()
}

// After (add):
select {
case w.hookChangedCh <- struct{}{}:
default: // channel already has a pending signal; coalesce
}
```

**Step 4: Fix the one existing test that sets onChange directly**

In `TestStatusFileWatcher_StopDuringDebounce`, the struct literal currently sets `onChange: func() {}`. Remove that field:
```go
// Remove this line from the struct literal:
onChange: func() {},
```
Add the channel:
```go
hookChangedCh: make(chan struct{}, 1),
```

**Step 5: Fix all callers of NewStatusFileWatcher**

Search for all calls:
```bash
grep -rn "NewStatusFileWatcher" /Users/mnicholson/code/github/hangar/.worktrees/json-http-hooks/
```

Update each call from `session.NewStatusFileWatcher(nil)` or `session.NewStatusFileWatcher(func(){...})` to `session.NewStatusFileWatcher()`.

Known locations (verify with grep above):
- `internal/ui/home.go` — three occurrences (lines ~773, ~793, ~4055)
- `internal/session/transition_daemon.go` — line ~204

**Step 6: Run all tests to verify nothing broken**

```bash
go test ./internal/session/... -v
```
Expected: all pass including new notify tests.

**Step 7: Commit**

```bash
git add internal/session/hook_watcher.go internal/session/hook_watcher_test.go \
        internal/ui/home.go internal/session/transition_daemon.go
git commit -m "feat(hooks): replace onChange callback with hookChangedCh notification channel"
```

---

### Task 2: Wire listenForHookChanges into the TUI

**Files:**
- Modify: `internal/ui/home.go`
- Modify: `internal/ui/update_handlers.go`

The TUI currently never gets a proactive wake-up when hooks fire. Add a `tea.Cmd` that blocks on the new channel and injects a `hookStatusChangedMsg` into Bubble Tea's event loop.

**Step 1: Add the message type and listener function**

In `internal/ui/home.go`, near the other message type declarations (around line 387):

```go
// hookStatusChangedMsg signals that a hook status file was processed.
// Triggers an immediate status refresh without waiting for the next tick.
type hookStatusChangedMsg struct{}
```

Near `listenForReloads` (around line 1294):

```go
// listenForHookChanges waits for a hook status change notification.
// MUST be re-issued in the Update handler to keep listening.
func listenForHookChanges(w *session.StatusFileWatcher) tea.Cmd {
	return func() tea.Msg {
		if w == nil {
			return nil
		}
		<-w.NotifyChannel()
		return hookStatusChangedMsg{}
	}
}
```

**Step 2: Write the test first**

In `internal/ui/` create `internal/ui/hook_listener_test.go`:

```go
package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sjoeboo/hangar/internal/session"
)

func TestListenForHookChanges_FiresOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)

	// Override hooks dir for test
	t.Setenv("HOME", tmpDir)

	watcher, err := session.NewStatusFileWatcher()
	if err != nil {
		t.Fatalf("NewStatusFileWatcher: %v", err)
	}
	go watcher.Start()
	defer watcher.Stop()

	// Write a hook file (triggers fsnotify → processFile → hookChangedCh)
	data, _ := json.Marshal(map[string]any{
		"status": "running", "session_id": "s1", "event": "UserPromptSubmit",
		"ts": time.Now().Unix(),
	})
	hookFile := filepath.Join(hooksDir, "inst-1.json")
	if err := os.WriteFile(hookFile, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// The tea.Cmd blocks until a notification arrives
	cmdFn := listenForHookChanges(watcher)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmdFn() }()

	select {
	case msg := <-done:
		if _, ok := msg.(hookStatusChangedMsg); !ok {
			t.Errorf("Expected hookStatusChangedMsg, got %T", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listenForHookChanges did not fire within 2s")
	}
}
```

Note: this test is intentionally simple — it verifies the plumbing. The full integration is tested manually.

**Step 3: Run test to confirm it compiles but listener is not yet connected**

```bash
go test ./internal/ui/... -run TestListenForHookChanges -v
```

**Step 4: Arm the listener in Init() and handle the message in Update()**

In `internal/ui/home.go`, in the `Init()` return commands section (around line 1275 where `listenForReloads` is armed):

```go
// Arm hook change listener if watcher is running
if h.hookWatcher != nil {
    cmds = append(cmds, listenForHookChanges(h.hookWatcher))
}
```

In `internal/ui/update_handlers.go`, find `handleStorageChanged` and add the new handler nearby:

```go
// handleHookStatusChanged applies fresh hook statuses to visible instances
// and re-arms the hook change listener. Called when a hook file is written.
func (h *Home) handleHookStatusChanged() tea.Cmd {
	// Apply hook statuses immediately to in-memory instances
	if h.hookWatcher != nil {
		h.instancesMu.RLock()
		for _, inst := range h.instances {
			if inst.Tool == "claude" || inst.Tool == "codex" {
				if hs := h.hookWatcher.GetHookStatus(inst.ID); hs != nil {
					inst.UpdateHookStatus(hs)
				}
			}
		}
		h.instancesMu.RUnlock()
		// Invalidate status count cache so View() picks up changes
		h.cachedStatusCounts.valid.Store(false)
	}
	// Re-arm the listener (must be re-issued every time, like listenForReloads)
	return listenForHookChanges(h.hookWatcher)
}
```

In `home.go` `Update()`, add the case near `storageChangedMsg` (around line 2298):

```go
case hookStatusChangedMsg:
    return h, h.handleHookStatusChanged()
```

**Step 5: Run all UI tests**

```bash
go test ./internal/ui/... -v 2>&1 | tail -30
```
Expected: all pass (including pre-existing failing tests which remain failing for known reasons per CLAUDE.md).

**Step 6: Commit**

```bash
git add internal/ui/home.go internal/ui/update_handlers.go internal/ui/hook_listener_test.go
git commit -m "feat(ui): wire listenForHookChanges for instant TUI refresh on hook events"
```

---

### Task 3: Move MapEventToStatus to session package

**Files:**
- Modify: `internal/session/hook_watcher.go` (add exported function)
- Modify: `cmd/hangar/hook_handler.go` (use session package function)
- Test: `internal/session/hook_watcher_test.go`

The HTTP server will need the same event→status mapping logic. Move it to the session package so it's shareable.

**Step 1: Write the test**

Add to `internal/session/hook_watcher_test.go`:

```go
func TestMapEventToStatus(t *testing.T) {
	tests := []struct {
		event  string
		want   string
	}{
		{"SessionStart", "waiting"},
		{"UserPromptSubmit", "running"},
		{"Stop", "waiting"},
		{"PermissionRequest", "waiting"},
		{"SessionEnd", "dead"},
		{"Notification", ""},
		{"UnknownEvent", ""},
	}
	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			got := MapEventToStatus(tt.event)
			if got != tt.want {
				t.Errorf("MapEventToStatus(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to confirm failure**

```bash
go test ./internal/session/... -run TestMapEventToStatus -v
```
Expected: compile error — `MapEventToStatus` not defined in session package.

**Step 3: Add exported function to session package**

Add to `internal/session/hook_watcher.go` (alongside `HookStatus`):

```go
// MapEventToStatus maps a Claude Code hook event name to a hangar status string.
// Returns empty string for events that don't change status.
func MapEventToStatus(event string) string {
	switch event {
	case "SessionStart":
		return "waiting"
	case "UserPromptSubmit":
		return "running"
	case "Stop":
		return "waiting"
	case "PermissionRequest":
		return "waiting"
	case "SessionEnd":
		return "dead"
	default:
		return ""
	}
}
```

**Step 4: Update hook_handler.go to use the session function**

In `cmd/hangar/hook_handler.go`, replace `mapEventToStatus` function body with a delegation:

```go
// mapEventToStatus delegates to the session package for the canonical mapping.
func mapEventToStatus(event string) string {
	return session.MapEventToStatus(event)
}
```

(Keep the local function so callers in the same file don't need changing.)

**Step 5: Run tests**

```bash
go test ./internal/session/... ./cmd/hangar/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```
Expected: all pass.

**Step 6: Commit**

```bash
git add internal/session/hook_watcher.go internal/session/hook_watcher_test.go cmd/hangar/hook_handler.go
git commit -m "refactor(hooks): export MapEventToStatus from session package for reuse"
```

---

### Task 4: Add hook server port to user config

**Files:**
- Modify: `internal/session/userconfig.go`
- Test: existing userconfig tests still pass (no new test needed — existing coverage sufficient)

**Step 1: Add the field to ClaudeSettings**

In `internal/session/userconfig.go`, in the `ClaudeSettings` struct (around line 402, after `HooksEnabled`):

```go
// HookServerPort is the TCP port for the embedded HTTP hook server.
// Claude Code 2.1.63+ can POST hook events directly to this server
// instead of spawning a subprocess for each event.
// Default: 2437. Set to 0 to disable HTTP hooks.
HookServerPort *int `toml:"hook_server_port"`
```

Add the accessor after `GetHooksEnabled`:

```go
// GetHookServerPort returns the HTTP hook server port, defaulting to 2437.
// Returns 0 if HTTP hooks are disabled via hook_server_port = 0.
func (c *ClaudeSettings) GetHookServerPort() int {
	if c.HookServerPort == nil {
		return 2437
	}
	return *c.HookServerPort
}
```

**Step 2: Run all tests to confirm nothing broken**

```bash
go test ./internal/session/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```
Expected: all pass.

**Step 3: Commit**

```bash
git add internal/session/userconfig.go
git commit -m "feat(config): add HookServerPort to ClaudeSettings (default 2437)"
```

---

### Task 5: Version detection for HTTP hook capability

**Files:**
- Modify: `internal/session/claude_hooks.go`
- Modify: `internal/session/claude_hooks_test.go`

Claude ≥ 2.1.63 supports `type: "http"` hooks. We need to detect the installed version and compare it.

**Step 1: Write the version parsing tests first**

Add to `internal/session/claude_hooks_test.go`:

```go
func TestParseClaudeVersion(t *testing.T) {
	tests := []struct {
		output  string
		wantVer string
		wantErr bool
	}{
		{"claude 2.1.63 (Claude Code)", "2.1.63", false},
		{"claude 1.0.0 (Claude Code)", "1.0.0", false},
		{"2.1.63", "2.1.63", false},
		{"Claude Code v2.1.63", "2.1.63", false},
		{"no version here", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			got, err := parseClaudeVersion(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseClaudeVersion(%q): expected error, got %q", tt.output, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseClaudeVersion(%q): unexpected error: %v", tt.output, err)
			}
			if got != tt.wantVer {
				t.Errorf("parseClaudeVersion(%q) = %q, want %q", tt.output, got, tt.wantVer)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version  string
		major    int
		minor    int
		patch    int
		want     bool
	}{
		{"2.1.63", 2, 1, 63, true},
		{"2.1.64", 2, 1, 63, true},
		{"2.2.0", 2, 1, 63, true},
		{"3.0.0", 2, 1, 63, true},
		{"2.1.62", 2, 1, 63, false},
		{"2.0.99", 2, 1, 63, false},
		{"1.9.99", 2, 1, 63, false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := versionAtLeast(tt.version, tt.major, tt.minor, tt.patch)
			if got != tt.want {
				t.Errorf("versionAtLeast(%q, %d, %d, %d) = %v, want %v",
					tt.version, tt.major, tt.minor, tt.patch, got, tt.want)
			}
		})
	}
}

func TestClaudeSupportsHTTPHooks(t *testing.T) {
	if claudeSupportsHTTPHooks("2.1.63") != true {
		t.Error("2.1.63 should support HTTP hooks")
	}
	if claudeSupportsHTTPHooks("2.1.62") != false {
		t.Error("2.1.62 should not support HTTP hooks")
	}
	if claudeSupportsHTTPHooks("") != false {
		t.Error("empty version should not support HTTP hooks")
	}
}
```

**Step 2: Run tests to confirm they fail**

```bash
go test ./internal/session/... -run "TestParseClaudeVersion|TestVersionAtLeast|TestClaudeSupportsHTTPHooks" -v
```
Expected: compile error — functions not defined yet.

**Step 3: Implement version detection functions**

Add to `internal/session/claude_hooks.go` (after the imports, before `InjectClaudeHooks`):

```go
import (
    // add to existing imports:
    "fmt"
    "os/exec"
    "regexp"
    "strconv"
)

var versionRegexp = regexp.MustCompile(`\b(\d+)\.(\d+)\.(\d+)\b`)

// parseClaudeVersion extracts the semver string from `claude --version` output.
func parseClaudeVersion(output string) (string, error) {
	m := versionRegexp.FindStringSubmatch(strings.TrimSpace(output))
	if m == nil {
		return "", fmt.Errorf("no semver found in %q", output)
	}
	return m[1] + "." + m[2] + "." + m[3], nil
}

// versionAtLeast reports whether version string (e.g. "2.1.63") is >= major.minor.patch.
func versionAtLeast(version string, major, minor, patch int) bool {
	m := versionRegexp.FindStringSubmatch(version)
	if m == nil {
		return false
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	if maj != major {
		return maj > major
	}
	if min != minor {
		return min > minor
	}
	return pat >= patch
}

// claudeSupportsHTTPHooks reports whether the given version supports type:"http" hooks.
// HTTP hooks were introduced in Claude Code 2.1.63.
func claudeSupportsHTTPHooks(version string) bool {
	return versionAtLeast(version, 2, 1, 63)
}

// DetectClaudeVersion runs `claude --version` and returns the parsed semver string.
// Returns empty string and an error if the version cannot be determined.
func DetectClaudeVersion() (string, error) {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("claude --version: %w", err)
	}
	return parseClaudeVersion(string(out))
}
```

**Step 4: Run tests**

```bash
go test ./internal/session/... -run "TestParseClaudeVersion|TestVersionAtLeast|TestClaudeSupportsHTTPHooks" -v
```
Expected: all pass.

**Step 5: Run full session tests**

```bash
go test ./internal/session/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```

**Step 6: Commit**

```bash
git add internal/session/claude_hooks.go internal/session/claude_hooks_test.go
git commit -m "feat(hooks): add claude version detection and HTTP hook capability check"
```

---

### Task 6: HTTP hook server package

**Files:**
- Create: `internal/hookserver/server.go`
- Create: `internal/hookserver/server_test.go`

**Step 1: Write the tests first**

Create `internal/hookserver/server_test.go`:

```go
package hookserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sjoeboo/hangar/internal/hookserver"
	"github.com/sjoeboo/hangar/internal/session"
)

func newTestWatcher(t *testing.T) *session.StatusFileWatcher {
	t.Helper()
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	_ = os.MkdirAll(hooksDir, 0755)
	// Override HOME so the watcher uses our temp dir
	t.Setenv("HOME", tmpDir)
	w, err := session.NewStatusFileWatcher()
	if err != nil {
		t.Fatalf("NewStatusFileWatcher: %v", err)
	}
	return w
}

func TestHookServer_ValidPayload(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher) // port 0 = use handler directly

	payload := map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hangar-Instance-Id", "inst-test-001")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	// Status should be in watcher
	time.Sleep(10 * time.Millisecond) // allow async processing if any
	hs := watcher.GetHookStatus("inst-test-001")
	if hs == nil {
		t.Fatal("Expected hook status to be set")
	}
	if hs.Status != "running" {
		t.Errorf("Status = %q, want running", hs.Status)
	}
}

func TestHookServer_MissingInstanceID(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

	payload := map[string]any{
		"hook_event_name": "Stop",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Hangar-Instance-Id header
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	// Must still return 200 (never block Claude)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even for missing instance ID", rr.Code)
	}
}

func TestHookServer_WrongMethod(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

	req := httptest.NewRequest(http.MethodGet, "/hooks", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHookServer_UnknownEvent(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

	payload := map[string]any{
		"hook_event_name": "SomeUnknownEvent",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-unknown")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	// Unknown events don't update status
	if watcher.GetHookStatus("inst-unknown") != nil {
		t.Error("Unknown event should not set hook status")
	}
}

func TestHookServer_NotifiesWatcher(t *testing.T) {
	watcher := newTestWatcher(t)
	srv := hookserver.New(0, watcher)

	payload := map[string]any{
		"hook_event_name": "Stop",
		"session_id":      "claude-sess-abc",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewReader(body))
	req.Header.Set("X-Hangar-Instance-Id", "inst-notify")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	select {
	case <-watcher.NotifyChannel():
		// correct — notification sent to TUI
	default:
		t.Fatal("Expected notification on watcher channel after hook POST")
	}
}
```

**Step 2: Run tests to confirm they fail**

```bash
go test ./internal/hookserver/... -v
```
Expected: compile error — package doesn't exist yet.

**Step 3: Implement the hook server**

Create `internal/hookserver/server.go`:

```go
package hookserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sjoeboo/hangar/internal/session"
)

// HookServer is an embedded HTTP server that receives Claude Code webhook events.
// It binds to 127.0.0.1 only and is lifecycle-bound to the TUI process.
type HookServer struct {
	port    int
	watcher *session.StatusFileWatcher
	server  *http.Server
}

// New creates a new HookServer. port=0 is valid for tests (use ServeHTTP directly).
func New(port int, watcher *session.StatusFileWatcher) *HookServer {
	s := &HookServer{
		port:    port,
		watcher: watcher,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/hooks", s.handleHook)
	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

// ServeHTTP implements http.Handler for testing — delegates directly to the mux.
func (s *HookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.Handler.ServeHTTP(w, r)
}

// Start binds to 127.0.0.1:{port} and begins serving. Blocks until ctx is cancelled.
// Returns nil on clean shutdown.
func (s *HookServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return fmt.Errorf("hookserver listen :%d: %w", s.port, err)
	}
	slog.Info("hookserver_started", slog.Int("port", s.port))

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

// hookPayload is the JSON body Claude Code sends for HTTP hook events.
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Matcher       string `json:"matcher,omitempty"`
}

func (s *HookServer) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	instanceID := r.Header.Get("X-Hangar-Instance-Id")
	if instanceID == "" {
		// Not a hangar-managed session — ignore silently, always 200
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64KB max
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload hookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := session.MapEventToStatus(payload.HookEventName)

	// Notification events: only "waiting" for permission_prompt/elicitation_dialog
	if payload.HookEventName == "Notification" {
		if payload.Matcher == "permission_prompt" || payload.Matcher == "elicitation_dialog" {
			status = "waiting"
		}
	}

	if status != "" {
		s.watcher.Notify(instanceID, status, payload.SessionID, payload.HookEventName)
	}

	w.WriteHeader(http.StatusOK)
}
```

You'll also need to add a `Notify` method to `StatusFileWatcher` in `internal/session/hook_watcher.go` that:
1. Updates the in-memory `statuses` map directly (same as `processFile` but from a struct)
2. Writes the status file atomically (same format as `writeHookStatus` in `hook_handler.go`)
3. Sends to `hookChangedCh`

```go
// Notify updates the hook status for an instance, writes the status file,
// and signals the TUI via hookChangedCh. Called by the HTTP hook server.
func (w *StatusFileWatcher) Notify(instanceID, status, sessionID, event string) {
	if instanceID == "" || status == "" {
		return
	}

	hs := &HookStatus{
		Status:    status,
		SessionID: sessionID,
		Event:     event,
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.statuses[instanceID] = hs
	w.mu.Unlock()

	// Write status file for startup catchup
	writeHookStatusFile(instanceID, status, sessionID, event, w.hooksDir)

	// Wake the TUI
	select {
	case w.hookChangedCh <- struct{}{}:
	default:
	}

	hookLog.Debug("http_hook_received",
		slog.String("instance", instanceID),
		slog.String("status", status),
		slog.String("event", event),
	)
}
```

Also extract the file-writing logic from `hook_handler.go` into `internal/session/` as:

```go
// writeHookStatusFile atomically writes a hook status JSON file to hooksDir.
func writeHookStatusFile(instanceID, status, sessionID, event, hooksDir string) {
	if instanceID == "" || status == "" || hooksDir == "" {
		return
	}
	type hookStatusFile struct {
		Status    string `json:"status"`
		SessionID string `json:"session_id,omitempty"`
		Event     string `json:"event"`
		Timestamp int64  `json:"ts"`
	}
	sf := hookStatusFile{
		Status:    status,
		SessionID: sessionID,
		Event:     event,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(sf)
	if err != nil {
		return
	}
	filePath := filepath.Join(hooksDir, instanceID+".json")
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmpPath, filePath)
}
```

**Step 4: Run hookserver tests**

```bash
go test ./internal/hookserver/... -v
```
Expected: all pass.

**Step 5: Run all tests**

```bash
go test ./... 2>&1 | grep -E "ok|FAIL"
```
Expected: all green.

**Step 6: Commit**

```bash
git add internal/hookserver/ internal/session/hook_watcher.go
git commit -m "feat(hookserver): add embedded HTTP hook server for Claude Code 2.1.63+"
```

---

### Task 7: Version-aware hook injection

**Files:**
- Modify: `internal/session/claude_hooks.go`
- Modify: `internal/session/claude_hooks_test.go`
- Modify: `cmd/hangar/hook_handler.go` (update callers)

**Step 1: Write the new tests first**

Add to `internal/session/claude_hooks_test.go`:

```go
func TestInjectClaudeHooks_CommandType_WhenPortZero(t *testing.T) {
	tmpDir := t.TempDir()

	installed, err := InjectClaudeHooks(tmpDir, 0)
	if err != nil {
		t.Fatalf("InjectClaudeHooks: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be installed")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Type == "http" {
				t.Error("Should not inject HTTP hook when port=0")
			}
		}
	}
}

func TestInjectClaudeHooks_HTTPType_WhenPortNonZero(t *testing.T) {
	tmpDir := t.TempDir()

	installed, err := InjectClaudeHooks(tmpDir, 2437)
	if err != nil {
		t.Fatalf("InjectClaudeHooks: %v", err)
	}
	if !installed {
		t.Error("Expected hooks to be installed")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	foundHTTP := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Type == "http" {
				foundHTTP = true
				if h.URL != "http://127.0.0.1:2437/hooks" {
					t.Errorf("HTTP hook URL = %q, want http://127.0.0.1:2437/hooks", h.URL)
				}
			}
		}
	}
	if !foundHTTP {
		t.Error("Expected HTTP hook entry when port=2437")
	}
}

func TestInjectClaudeHooks_UpgradeCommandToHTTP(t *testing.T) {
	tmpDir := t.TempDir()

	// First: install command hooks
	if _, err := InjectClaudeHooks(tmpDir, 0); err != nil {
		t.Fatalf("command install: %v", err)
	}

	// Now: upgrade to HTTP hooks
	upgraded, err := InjectClaudeHooks(tmpDir, 2437)
	if err != nil {
		t.Fatalf("http upgrade: %v", err)
	}
	if !upgraded {
		t.Error("Expected upgrade to return installed=true")
	}

	// Verify command hooks removed, HTTP hooks present
	data, _ := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	var settings map[string]json.RawMessage
	_ = json.Unmarshal(data, &settings)
	var hooks map[string]json.RawMessage
	_ = json.Unmarshal(settings["hooks"], &hooks)
	var matchers []claudeHookMatcher
	_ = json.Unmarshal(hooks["SessionStart"], &matchers)

	foundCommand := false
	foundHTTP := false
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Command == hangarHookCommand {
				foundCommand = true
			}
			if h.Type == "http" {
				foundHTTP = true
			}
		}
	}
	if foundCommand {
		t.Error("Command hook should be removed after HTTP upgrade")
	}
	if !foundHTTP {
		t.Error("HTTP hook should be present after upgrade")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/session/... -run "TestInjectClaudeHooks_CommandType|TestInjectClaudeHooks_HTTPType|TestInjectClaudeHooks_Upgrade" -v
```
Expected: compile error — `InjectClaudeHooks` signature mismatch.

**Step 3: Update the InjectClaudeHooks signature and logic**

In `internal/session/claude_hooks.go`:

1. Add to the structs: `URL`, `Headers`, `AllowedEnvVars` fields to `claudeHookEntry`:

```go
type claudeHookEntry struct {
	Type           string            `json:"type"`
	Command        string            `json:"command,omitempty"`
	Async          bool              `json:"async,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	AllowedEnvVars []string          `json:"allowedEnvVars,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
}
```

2. Add a new helper:

```go
const hangarHookURL = "http://127.0.0.1:%d/hooks"

func hangarHTTPHook(port int) claudeHookEntry {
	return claudeHookEntry{
		Type:    "http",
		URL:     fmt.Sprintf(hangarHookURL, port),
		Headers: map[string]string{"X-Hangar-Instance-Id": "$HANGAR_INSTANCE_ID"},
		AllowedEnvVars: []string{"HANGAR_INSTANCE_ID"},
		Timeout: 5,
	}
}
```

3. Change `InjectClaudeHooks` signature to accept port:

```go
func InjectClaudeHooks(configDir string, port int) (bool, error)
```

4. Update `hangarHook()` selection logic inside — when `port > 0`, use `hangarHTTPHook(port)`, otherwise use existing `hangarHook()`.

5. Update `hooksAlreadyInstalled` and `eventHasHangarHook` to check for EITHER command hook OR HTTP hook (matching `127.0.0.1:*/hooks`).

6. Add upgrade logic: before the "check if already installed" gate, check if command hooks exist but port > 0 (HTTP wanted). If so, remove command hooks first, then proceed with HTTP injection.

**Step 4: Update all callers to pass port**

In `cmd/hangar/hook_handler.go`:
```go
// handleHooksInstall uses port=0 (command hooks) when called from CLI
// because the HTTP server isn't running in this context.
installed, err := session.InjectClaudeHooks(configDir, 0)
```

Update existing test helpers in `claude_hooks_test.go` to pass 0:
```go
// All existing tests that call InjectClaudeHooks(tmpDir) → InjectClaudeHooks(tmpDir, 0)
```

**Step 5: Update RemoveClaudeHooks and CheckClaudeHooksInstalled**

`eventHasHangarHook` needs to match HTTP entries too:

```go
func eventHasHangarHook(raw json.RawMessage) bool {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return false
	}
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, hangarHookCommand) {
				return true
			}
			// Match HTTP hook by URL pattern
			if h.Type == "http" && strings.Contains(h.URL, "/hooks") && strings.Contains(h.URL, "127.0.0.1") {
				return true
			}
		}
	}
	return false
}
```

`removeHangarFromEvent` needs to remove HTTP entries too:

```go
// In the hook filtering loop, also match HTTP hooks:
if h.Type == "http" && strings.Contains(h.URL, "127.0.0.1") && strings.Contains(h.URL, "/hooks") {
    removed = true
    continue
}
```

**Step 6: Run all tests**

```bash
go test ./... 2>&1 | grep -E "ok|FAIL"
```
Expected: all green.

**Step 7: Commit**

```bash
git add internal/session/claude_hooks.go internal/session/claude_hooks_test.go cmd/hangar/hook_handler.go
git commit -m "feat(hooks): version-aware injection — HTTP hooks for Claude >= 2.1.63, command fallback"
```

---

### Task 8: TUI wiring — start HTTP server and upgrade hooks on startup

**Files:**
- Modify: `internal/ui/home.go`
- Modify: `cmd/hangar/main.go` (minimal — just ensure hookserver import is available if needed)

**Step 1: Start HTTP server in Home.Init()**

In `internal/ui/home.go`, in the hooks initialization block (around line 766 where `hooksEnabled` is checked):

After the hookWatcher is started, start the HTTP server:

```go
// Start embedded HTTP hook server if port is configured
port := 0
if userConfig != nil {
    port = userConfig.Claude.GetHookServerPort()
}
if hooksEnabled && port > 0 {
    srv := hookserver.New(port, h.hookWatcher)
    h.hookServer = srv
    go func() {
        if err := srv.Start(h.ctx); err != nil {
            uiLog.Warn("hookserver_failed", slog.String("error", err.Error()))
        }
    }()
}
```

Add `hookServer *hookserver.HookServer` field to the `Home` struct.

Add the import: `"github.com/sjoeboo/hangar/internal/hookserver"`

**Step 2: Upgrade hooks on startup**

After the hook server is started (still in Init), call:

```go
// Silently upgrade command hooks → HTTP hooks if version now supports it
if hooksEnabled && port > 0 {
    go func() {
        if upgraded, err := session.InjectClaudeHooks(configDir, port); err != nil {
            uiLog.Warn("hook_upgrade_failed", slog.String("error", err.Error()))
        } else if upgraded {
            uiLog.Info("hooks_upgraded_to_http", slog.Int("port", port))
        }
    }()
}
```

The `InjectClaudeHooks` is idempotent — if HTTP hooks are already installed, it returns `false, nil`. If command hooks are present and we want HTTP, the upgrade path (Task 7 step 3.6) handles the replacement.

**Step 3: Run all tests**

```bash
go test ./... -race 2>&1 | grep -E "ok|FAIL"
```
Expected: all green, no races.

**Step 4: Build and smoke test**

```bash
go build ./cmd/hangar/...
```
Expected: builds cleanly.

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): start HTTP hook server on TUI launch, upgrade hooks on startup"
```

---

### Task 9: Update `hangar hooks status` output

**Files:**
- Modify: `cmd/hangar/hook_handler.go`

**Step 1: Update handleHooksStatus to report hook type**

In `handleHooksStatus()`, after the installed check, show which type is active:

```go
if installed {
    fmt.Println("Status: INSTALLED")
    fmt.Printf("Config: %s/settings.json\n", configDir)
    // Detect which type
    if session.CheckClaudeHTTPHooksInstalled(configDir) {
        fmt.Println("Type: HTTP (Claude >= 2.1.63)")
    } else {
        fmt.Println("Type: command (hangar hook-handler)")
    }
} else {
    fmt.Println("Status: NOT INSTALLED")
    fmt.Println("Run 'hangar hooks install' to install.")
}
```

Add `CheckClaudeHTTPHooksInstalled(configDir string) bool` to `internal/session/claude_hooks.go`:

```go
// CheckClaudeHTTPHooksInstalled returns true if HTTP hooks (not command hooks) are installed.
func CheckClaudeHTTPHooksInstalled(configDir string) bool {
    // Read settings.json and check for type:"http" hangar entries
    // (implementation mirrors CheckClaudeHooksInstalled but checks h.Type == "http")
}
```

**Step 2: Run all tests, build, commit**

```bash
go test ./... 2>&1 | grep -E "ok|FAIL"
go build ./cmd/hangar/...
git add cmd/hangar/hook_handler.go internal/session/claude_hooks.go
git commit -m "feat(cli): hangar hooks status shows http vs command hook type"
```

---

## Final verification

```bash
go test ./... -race
go build ./cmd/hangar/...
```

All tests should pass. No races. Binary builds cleanly.

Manual smoke test checklist:
- [ ] `hangar hooks install` installs HTTP hooks when Claude ≥ 2.1.63
- [ ] `hangar hooks status` shows `Type: HTTP`
- [ ] Claude session status changes appear in TUI within ~200ms (vs ~4s before)
- [ ] `hangar hooks uninstall` removes HTTP hooks cleanly
- [ ] Old Claude (simulate with port=0): command hooks injected as before
