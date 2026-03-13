package pr

import (
	"testing"
	"time"
)

func TestSetSessionPR_AutoEvictsMergedClosed(t *testing.T) {
	m := New()

	// OPEN and DRAFT should be stored normally
	m.SetSessionPR("open-1", &PR{State: StateOpen, URL: "https://example.com/1"})
	m.SetSessionPR("draft-1", &PR{State: StateDraft, URL: "https://example.com/2"})
	m.SetSessionPR("nil-1", nil)

	if got := len(m.GetAll()); got != 2 {
		t.Fatalf("GetAll after open+draft: want 2, got %d", got)
	}

	// MERGED and CLOSED should be auto-evicted (not stored)
	m.SetSessionPR("merged-1", &PR{State: StateMerged, URL: "https://example.com/3"})
	m.SetSessionPR("closed-1", &PR{State: StateClosed, URL: "https://example.com/4"})

	if got := len(m.GetAll()); got != 2 {
		t.Fatalf("GetAll after merged+closed: want 2 (unchanged), got %d", got)
	}

	// Verify MERGED/CLOSED entries don't exist in sessionFetched
	if _, exists := m.SessionPRStaleAt("merged-1"); exists {
		t.Error("merged-1 should not have a sessionFetched entry")
	}
	if _, exists := m.SessionPRStaleAt("closed-1"); exists {
		t.Error("closed-1 should not have a sessionFetched entry")
	}
}

func TestSetSessionPR_EvictsExistingOnStateChange(t *testing.T) {
	m := New()

	// Store an OPEN PR, then update to MERGED — should evict
	m.SetSessionPR("s1", &PR{State: StateOpen, URL: "https://example.com/1"})
	if _, exists := m.GetSessionPR("s1"); !exists {
		t.Fatal("s1 should exist after setting OPEN")
	}

	m.SetSessionPR("s1", &PR{State: StateMerged, URL: "https://example.com/1"})
	if _, exists := m.GetSessionPR("s1"); exists {
		t.Error("s1 should be evicted after state changed to MERGED")
	}
}

func TestClearClosedSessionPRs(t *testing.T) {
	m := New()

	// Seed session PRs with mixed states via direct map access
	// (SetSessionPR auto-evicts MERGED/CLOSED, so we bypass it for this test)
	m.mu.Lock()
	m.sessionPRs["open-1"] = &PR{State: StateOpen, URL: "https://example.com/1"}
	m.sessionFetched["open-1"] = time.Now()
	m.sessionPRs["merged-1"] = &PR{State: StateMerged, URL: "https://example.com/2"}
	m.sessionFetched["merged-1"] = time.Now()
	m.sessionPRs["closed-1"] = &PR{State: StateClosed, URL: "https://example.com/3"}
	m.sessionFetched["closed-1"] = time.Now()
	m.sessionPRs["draft-1"] = &PR{State: StateDraft, URL: "https://example.com/4"}
	m.sessionFetched["draft-1"] = time.Now()
	m.sessionPRs["nil-1"] = nil
	m.sessionFetched["nil-1"] = time.Now()
	m.mu.Unlock()

	if got := len(m.GetAll()); got != 4 {
		t.Fatalf("GetAll before clear: want 4, got %d", got)
	}

	removed := m.ClearClosedSessionPRs()
	if removed != 2 {
		t.Fatalf("ClearClosedSessionPRs: want 2 removed, got %d", removed)
	}

	all := m.GetAll()
	for _, p := range all {
		if p.State == StateMerged || p.State == StateClosed {
			t.Errorf("found %s PR after clear: %s", p.State, p.URL)
		}
	}

	// open + draft should remain
	if got := len(all); got != 2 {
		t.Errorf("GetAll after clear: want 2, got %d", got)
	}

	// Verify sessionFetched entries are also cleaned up
	if _, exists := m.SessionPRStaleAt("merged-1"); exists {
		t.Error("sessionFetched entry for merged-1 should be removed")
	}
	if _, exists := m.SessionPRStaleAt("open-1"); !exists {
		t.Error("sessionFetched entry for open-1 should still exist")
	}
}

func TestClearClosedSessionPRs_Empty(t *testing.T) {
	m := New()
	removed := m.ClearClosedSessionPRs()
	if removed != 0 {
		t.Errorf("want 0 removed from empty manager, got %d", removed)
	}
	if got := len(m.GetAll()); got != 0 {
		t.Errorf("GetAll after empty clear: want 0, got %d", got)
	}
}

func TestClearClosedSessionPRs_OnlyOpen(t *testing.T) {
	m := New()
	m.SetSessionPR("s1", &PR{State: StateOpen, URL: "https://example.com/1"})
	m.SetSessionPR("s2", &PR{State: StateDraft, URL: "https://example.com/2"})

	removed := m.ClearClosedSessionPRs()
	if removed != 0 {
		t.Errorf("want 0 removed (all open/draft), got %d", removed)
	}
	if got := len(m.GetAll()); got != 2 {
		t.Errorf("want 2 remaining, got %d", got)
	}
}

func TestClearClosedSessionPRs_PreservesNilEntries(t *testing.T) {
	m := New()
	m.SetSessionPR("no-pr", nil)

	removed := m.ClearClosedSessionPRs()
	if removed != 0 {
		t.Errorf("want 0 removed, got %d", removed)
	}
	if _, exists := m.SessionPRStaleAt("no-pr"); !exists {
		t.Error("nil entry's sessionFetched should be preserved")
	}
}

func TestClearClosedSessionPRs_NotifiesOnChange(t *testing.T) {
	m := New()

	// Seed directly to bypass auto-evict
	m.mu.Lock()
	m.sessionPRs["merged"] = &PR{State: StateMerged, URL: "https://example.com/1"}
	m.sessionFetched["merged"] = time.Now()
	m.mu.Unlock()

	notified := make(chan struct{}, 1)
	m.RegisterOnChange(func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	})

	m.ClearClosedSessionPRs()

	select {
	case <-notified:
		// good — notifyChange fired
	default:
		t.Error("onChange callback not invoked synchronously after clearing closed PRs")
	}
}

func TestClearClosedSessionPRs_NoNotifyWhenNothingRemoved(t *testing.T) {
	m := New()
	m.SetSessionPR("s1", &PR{State: StateOpen, URL: "https://example.com/1"})

	notified := make(chan struct{}, 1)
	m.RegisterOnChange(func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	})
	// Drain the notify from SetSessionPR
	select {
	case <-notified:
	default:
	}

	m.ClearClosedSessionPRs()

	select {
	case <-notified:
		t.Error("onChange should not fire when nothing was removed")
	case <-time.After(50 * time.Millisecond):
		// correct: no notification
	}
}
