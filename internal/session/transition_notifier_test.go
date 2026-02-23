package session

import (
	"testing"
	"time"
)

func TestShouldNotifyTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{name: "running to waiting", from: "running", to: "waiting", want: true},
		{name: "running to error", from: "running", to: "error", want: true},
		{name: "running to idle", from: "running", to: "idle", want: true},
		{name: "waiting to running", from: "waiting", to: "running", want: false},
		{name: "same status", from: "running", to: "running", want: false},
		{name: "empty from", from: "", to: "waiting", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldNotifyTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("ShouldNotifyTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestChoosePollInterval(t *testing.T) {
	if got := choosePollInterval(map[string]string{"a": "running"}); got != notifyPollFast {
		t.Fatalf("running interval = %v, want %v", got, notifyPollFast)
	}
	if got := choosePollInterval(map[string]string{"a": "waiting"}); got != notifyPollMedium {
		t.Fatalf("waiting interval = %v, want %v", got, notifyPollMedium)
	}
	if got := choosePollInterval(map[string]string{"a": "idle"}); got != notifyPollSlow {
		t.Fatalf("idle interval = %v, want %v", got, notifyPollSlow)
	}
}

func TestTerminalHookTransitionCandidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		tool string
		hs   *HookStatus
		want bool
	}{
		{
			name: "claude stop terminal",
			tool: "claude",
			hs: &HookStatus{
				Status:    "waiting",
				Event:     "Stop",
				UpdatedAt: now,
			},
			want: true,
		},
		{
			name: "claude session start ignored",
			tool: "claude",
			hs: &HookStatus{
				Status:    "waiting",
				Event:     "SessionStart",
				UpdatedAt: now,
			},
			want: false,
		},
		{
			name: "codex turn complete terminal",
			tool: "codex",
			hs: &HookStatus{
				Status:    "waiting",
				Event:     "agent-turn-complete",
				UpdatedAt: now,
			},
			want: true,
		},
		{
			name: "codex turn start ignored",
			tool: "codex",
			hs: &HookStatus{
				Status:    "running",
				Event:     "agent-turn-start",
				UpdatedAt: now,
			},
			want: false,
		},
		{
			name: "stale hook ignored",
			tool: "codex",
			hs: &HookStatus{
				Status:    "waiting",
				Event:     "agent-turn-complete",
				UpdatedAt: now.Add(-2 * time.Minute),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := terminalHookTransitionCandidate(tt.tool, tt.hs)
			if got != tt.want {
				t.Fatalf("terminalHookTransitionCandidate(%q, %+v) = %v, want %v", tt.tool, tt.hs, got, tt.want)
			}
		})
	}
}

func TestIsCodexTerminalHookEvent(t *testing.T) {
	if !isCodexTerminalHookEvent("agent-turn-complete") {
		t.Fatal("expected terminal event to match")
	}
	if !isCodexTerminalHookEvent("turn/failed") {
		t.Fatal("expected failed event to match")
	}
	if isCodexTerminalHookEvent("thread.started") {
		t.Fatal("thread.started should not be terminal")
	}
}
