package ui

import (
	"strings"
	"testing"
	"time"
)

func TestRenderSessionList_ShowsPendingWorktree(t *testing.T) {
	h := NewHome()
	h.width = 80
	h.height = 20
	h.pendingWorktrees = []pendingWorktreeItem{
		{branchName: "feat/my-branch", groupPath: "default", startedAt: time.Now()},
	}
	output := h.renderSessionList(80, 20)
	if !strings.Contains(output, "Creating worktree: feat/my-branch") {
		t.Errorf("expected ghost row 'Creating worktree: feat/my-branch' in session list, got:\n%s", output)
	}
}
