package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfirmDialog_BulkDeleteSessions(t *testing.T) {
	d := NewConfirmDialog()
	d.SetSize(80, 24)

	d.ShowBulkDeleteSessions(
		[]string{"id-1", "id-2", "id-3"},
		[]string{"session-alpha", "session-beta [worktree]", "session-gamma"},
	)

	if !d.IsVisible() {
		t.Fatal("dialog should be visible after ShowBulkDeleteSessions")
	}
	if d.GetConfirmType() != ConfirmBulkDeleteSessions {
		t.Errorf("confirm type = %v, want ConfirmBulkDeleteSessions", d.GetConfirmType())
	}
	ids := d.GetTargetIDs()
	if len(ids) != 3 {
		t.Fatalf("GetTargetIDs len = %d, want 3", len(ids))
	}
	if ids[0] != "id-1" || ids[1] != "id-2" || ids[2] != "id-3" {
		t.Errorf("GetTargetIDs = %v, want [id-1 id-2 id-3]", ids)
	}

	view := d.View()
	if !strings.Contains(view, "3 sessions") {
		t.Error("view should mention '3 sessions'")
	}
	if !strings.Contains(view, "session-alpha") {
		t.Error("view should list session names")
	}
	if !strings.Contains(view, "session-beta [worktree]") {
		t.Error("view should show worktree annotation")
	}
}

func TestConfirmDialog_BulkDeleteSessions_TruncatesLongList(t *testing.T) {
	d := NewConfirmDialog()
	d.SetSize(80, 40)

	ids := make([]string, 12)
	names := make([]string, 12)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
		names[i] = fmt.Sprintf("session-%d", i)
	}
	d.ShowBulkDeleteSessions(ids, names)

	view := d.View()
	if !strings.Contains(view, "and 4 more") {
		t.Error("view should truncate list and show '… and 4 more'")
	}
}

func TestConfirmDialog_BulkRestart(t *testing.T) {
	d := NewConfirmDialog()
	d.SetSize(80, 24)

	d.ShowBulkRestart([]string{"id-1", "id-2", "id-3", "id-4", "id-5"})

	if !d.IsVisible() {
		t.Fatal("dialog should be visible after ShowBulkRestart")
	}
	if d.GetConfirmType() != ConfirmBulkRestart {
		t.Errorf("confirm type = %v, want ConfirmBulkRestart", d.GetConfirmType())
	}

	view := d.View()
	if !strings.Contains(view, "5 sessions") {
		t.Error("view should mention '5 sessions'")
	}
}

func TestConfirmDialog_HideClears_BulkFields(t *testing.T) {
	d := NewConfirmDialog()
	d.ShowBulkDeleteSessions([]string{"id-1"}, []string{"sess-1"})
	d.Hide()

	if d.IsVisible() {
		t.Error("dialog should not be visible after Hide")
	}
	if len(d.GetTargetIDs()) != 0 {
		t.Error("targetIDs should be cleared after Hide")
	}
}
