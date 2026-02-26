package ui

import (
	"strings"
	"testing"
)

func TestWorktreeFinishDialog_SetPR_NoPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(nil, true) // loaded but no PR

	view := d.View()
	if !strings.Contains(view, "No PR") {
		t.Errorf("expected 'No PR' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_Checking(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	// prLoaded=false = still checking (default after Show)
	view := d.View()
	if !strings.Contains(view, "checking") {
		t.Errorf("expected 'checking' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_WithPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(&prCacheEntry{
		Number:       42,
		Title:        "Add user auth",
		State:        "OPEN",
		ChecksPassed: 5,
		ChecksFailed: 1,
		HasChecks:    true,
	}, true)

	view := d.View()
	if !strings.Contains(view, "#42") {
		t.Errorf("expected '#42' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Add user auth") {
		t.Errorf("expected title in view, got:\n%s", view)
	}
	if !strings.Contains(view, "OPEN") {
		t.Errorf("expected 'OPEN' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_SetPR_CILine(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetDirtyStatus(false)

	d.SetPR(&prCacheEntry{
		Number:        7,
		Title:         "Fix bug",
		State:         "OPEN",
		ChecksPassed:  3,
		ChecksFailed:  0,
		ChecksPending: 2,
		HasChecks:     true,
	}, true)

	view := d.View()
	// Should show passed and pending counts
	if !strings.Contains(view, "3") {
		t.Errorf("expected passed count '3' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "2") {
		t.Errorf("expected pending count '2' in view, got:\n%s", view)
	}
}

func TestWorktreeFinishDialog_ShowResetsPR(t *testing.T) {
	d := NewWorktreeFinishDialog()
	d.SetSize(120, 40)
	d.Show("id1", "my-session", "feat/foo", "/repo", "/repo/.worktrees/foo")
	d.SetPR(&prCacheEntry{Number: 1, State: "OPEN"}, true)

	// Re-show should reset PR state to "checking"
	d.Show("id2", "other", "feat/bar", "/repo", "/repo/.worktrees/bar")

	view := d.View()
	if !strings.Contains(view, "checking") {
		t.Errorf("expected 'checking' after re-show, got:\n%s", view)
	}
}
