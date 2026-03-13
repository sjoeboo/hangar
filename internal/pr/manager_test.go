package pr

import (
	"testing"
	"time"
)

func TestClearClosedSessionPRs(t *testing.T) {
	m := New()

	// Seed session PRs with mixed states
	m.SetSessionPR("open-1", &PR{State: "OPEN", URL: "https://example.com/1"})
	m.SetSessionPR("merged-1", &PR{State: "MERGED", URL: "https://example.com/2"})
	m.SetSessionPR("closed-1", &PR{State: "CLOSED", URL: "https://example.com/3"})
	m.SetSessionPR("draft-1", &PR{State: "DRAFT", URL: "https://example.com/4"})
	m.SetSessionPR("nil-1", nil) // no PR found for branch

	if got := len(m.GetAll()); got != 4 {
		t.Fatalf("GetAll before clear: want 4, got %d", got)
	}

	removed := m.ClearClosedSessionPRs()
	if removed != 2 {
		t.Fatalf("ClearClosedSessionPRs: want 2 removed, got %d", removed)
	}

	all := m.GetAll()
	for _, p := range all {
		if p.State == "MERGED" || p.State == "CLOSED" {
			t.Errorf("found %s PR after clear: %s", p.State, p.URL)
		}
	}

	// open + draft should remain
	if got := len(all); got != 2 {
		t.Errorf("GetAll after clear: want 2, got %d", got)
	}

	// Verify sessionFetched entries are also cleaned up
	_, exists := m.SessionPRStaleAt("merged-1")
	if exists {
		t.Error("sessionFetched entry for merged-1 should be removed")
	}
	_, exists = m.SessionPRStaleAt("open-1")
	if !exists {
		t.Error("sessionFetched entry for open-1 should still exist")
	}
}

func TestClearClosedSessionPRs_Empty(t *testing.T) {
	m := New()
	removed := m.ClearClosedSessionPRs()
	if removed != 0 {
		t.Errorf("want 0 removed from empty manager, got %d", removed)
	}
}

func TestClearClosedSessionPRs_OnlyOpen(t *testing.T) {
	m := New()
	m.SetSessionPR("s1", &PR{State: "OPEN", URL: "https://example.com/1"})
	m.SetSessionPR("s2", &PR{State: "DRAFT", URL: "https://example.com/2"})

	removed := m.ClearClosedSessionPRs()
	if removed != 0 {
		t.Errorf("want 0 removed (all open/draft), got %d", removed)
	}
	if got := len(m.GetAll()); got != 2 {
		t.Errorf("want 2 remaining, got %d", got)
	}
}

func TestClearClosedSessionPRs_NotifiesOnChange(t *testing.T) {
	m := New()
	m.SetSessionPR("merged", &PR{State: "MERGED", URL: "https://example.com/1"})

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
		// good
	case <-time.After(time.Second):
		t.Error("onChange callback not invoked after clearing closed PRs")
	}
}
