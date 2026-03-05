package ui

import (
	"strings"
	"testing"
)

const sampleDiff = `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 package main

+// added comment
 func main() {}
diff --git a/bar.go b/bar.go
index 111..222 100644
--- a/bar.go
+++ b/bar.go
@@ -1,2 +1,1 @@
 package main
-// removed line
`

func TestDiffView_ParseAndSummary(t *testing.T) {
	dv := NewDiffView()
	if err := dv.Parse(sampleDiff); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if dv.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", dv.FileCount())
	}
	summary := dv.Summary()
	if !strings.Contains(summary, "2 file") {
		t.Errorf("expected '2 file' in summary, got %q", summary)
	}
}

func TestDiffView_EmptyDiff(t *testing.T) {
	dv := NewDiffView()
	if err := dv.Parse(""); err != nil {
		t.Fatalf("Parse of empty string failed: %v", err)
	}
	if dv.Summary() != "no changes" {
		t.Errorf("expected 'no changes', got %q", dv.Summary())
	}
}

func TestDiffView_IsVisible(t *testing.T) {
	dv := NewDiffView()
	if dv.IsVisible() {
		t.Error("new DiffView should not be visible")
	}
	dv.Show()
	if !dv.IsVisible() {
		t.Error("DiffView should be visible after Show()")
	}
	dv.Hide()
	if dv.IsVisible() {
		t.Error("DiffView should not be visible after Hide()")
	}
}

func TestDiffView_FileUnderCursor(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.SetSize(120, 40)
	// At scrollOffset 0, first file header should be "foo.go"
	path, line := dv.FileUnderCursor()
	if !strings.Contains(path, "foo.go") {
		t.Errorf("expected foo.go under cursor at top, got %q", path)
	}
	if line < 1 {
		t.Errorf("expected line >= 1, got %d", line)
	}
}

func TestDiffView_Scroll(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	// Expand all files so there are enough lines to scroll
	for i := range dv.diffFiles {
		dv.diffFiles[i].expanded = true
	}
	dv.SetSize(120, 10)
	dv.rebuildLines()

	dv.ScrollDown(3)
	if dv.scrollOffset < 1 {
		t.Errorf("ScrollDown should increase offset above 0, got %d (total lines: %d)", dv.scrollOffset, len(dv.lines))
	}

	before := dv.scrollOffset
	dv.ScrollUp(1)
	if dv.scrollOffset >= before {
		t.Errorf("ScrollUp should decrease offset, was %d now %d", before, dv.scrollOffset)
	}

	// Cannot scroll above 0
	dv.ScrollUp(999)
	if dv.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0 after large scroll up, got %d", dv.scrollOffset)
	}

	// ScrollDown(999) clamps at max
	dv2 := NewDiffView()
	_ = dv2.Parse(sampleDiff)
	for i := range dv2.diffFiles {
		dv2.diffFiles[i].expanded = true
	}
	dv2.SetSize(120, 20)
	dv2.rebuildLines()
	dv2.ScrollDown(999)
	visibleHeight := 16
	maxOffset := len(dv2.lines) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if dv2.scrollOffset != maxOffset {
		t.Errorf("expected scrollOffset clamped at %d, got %d", maxOffset, dv2.scrollOffset)
	}
}

func TestDiffView_HandleKey_Close(t *testing.T) {
	closeKeys := []string{"q", "esc", "D"}
	for _, key := range closeKeys {
		t.Run(key, func(t *testing.T) {
			dv := NewDiffView()
			_ = dv.Parse(sampleDiff)
			dv.Show()

			handled, cmd := dv.HandleKey(key)
			if !handled {
				t.Errorf("expected %q to be handled", key)
			}
			if cmd != nil {
				t.Errorf("expected nil cmd for %q, got non-nil", key)
			}
			if dv.IsVisible() {
				t.Errorf("expected DiffView to be hidden after %q", key)
			}
		})
	}
}

func TestDiffView_HandleKey_EditorNoFile(t *testing.T) {
	// 'e' key with no file under cursor (empty diff) — handled but no cmd.
	dv := NewDiffView()
	dv.SetSize(80, 24)
	dv.Show()
	// don't parse any diff — lines will be empty
	handled, cmd := dv.HandleKey("e")
	if !handled {
		t.Error("expected 'e' to be handled even with empty diff")
	}
	if cmd != nil {
		t.Error("expected nil cmd when no file under cursor")
	}
	if !dv.IsVisible() {
		t.Error("overlay should remain visible when no file under cursor")
	}
}

func TestDiffView_HandleKey_FileNavigation(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()
	dv.SetSize(120, 40)

	// Initially on first file
	if dv.diffFileCursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", dv.diffFileCursor)
	}

	// j moves to next file
	handled, _ := dv.HandleKey("j")
	if !handled {
		t.Error("expected j to be handled")
	}
	if dv.diffFileCursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", dv.diffFileCursor)
	}

	// j at last file does not go further
	dv.HandleKey("j")
	if dv.diffFileCursor != 1 {
		t.Errorf("expected cursor still at 1 (last file), got %d", dv.diffFileCursor)
	}

	// k moves back
	dv.HandleKey("k")
	if dv.diffFileCursor != 0 {
		t.Errorf("expected cursor back at 0 after k, got %d", dv.diffFileCursor)
	}

	// k at first file does not go negative
	dv.HandleKey("k")
	if dv.diffFileCursor != 0 {
		t.Errorf("expected cursor still at 0, got %d", dv.diffFileCursor)
	}
}

func TestDiffView_HandleKey_Toggle(t *testing.T) {
	dv := NewDiffView()
	_ = dv.Parse(sampleDiff)
	dv.Show()
	dv.SetSize(120, 40)

	// sampleDiff has 2 files — both start collapsed (autoExpand only for 1 file)
	if dv.diffFiles[0].expanded {
		t.Fatal("expected first file to start collapsed with 2-file diff")
	}

	// enter expands
	handled, _ := dv.HandleKey("enter")
	if !handled {
		t.Error("expected enter to be handled")
	}
	if !dv.diffFiles[0].expanded {
		t.Error("expected first file to be expanded after enter")
	}

	// enter again collapses
	dv.HandleKey("enter")
	if dv.diffFiles[0].expanded {
		t.Error("expected first file to be collapsed after second enter")
	}
}

func TestDiffView_SingleFile_AutoExpand(t *testing.T) {
	singleFileDiff := `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -1,2 +1,3 @@
 package main
+// added
 func main() {}
`
	dv := NewDiffView()
	_ = dv.Parse(singleFileDiff)
	if len(dv.diffFiles) != 1 {
		t.Fatalf("expected 1 file, got %d", len(dv.diffFiles))
	}
	if !dv.diffFiles[0].expanded {
		t.Error("single-file diff should auto-expand")
	}
}

func TestDiffView_HandleKey_PagerBindings(t *testing.T) {
	// height=24 → fullPage = 24-4 = 20, halfPage = 10
	setup := func() *DiffView {
		dv := NewDiffView()
		_ = dv.Parse(sampleDiff)
		dv.Show()
		// Expand all files to have enough lines for paging
		for i := range dv.diffFiles {
			dv.diffFiles[i].expanded = true
		}
		dv.SetSize(120, 24)
		dv.rebuildLines()
		// Scroll to middle so both up and down have room
		dv.ScrollDown(5)
		return dv
	}

	halfPage := 10

	t.Run("f full-page-down", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		handled, _ := dv.HandleKey("f")
		if !handled {
			t.Error("expected f to be handled")
		}
		if dv.scrollOffset < before {
			t.Errorf("f should scroll down; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("ctrl+f full-page-down", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("ctrl+f")
		if dv.scrollOffset < before {
			t.Errorf("ctrl+f should scroll down; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("pgdown full-page-down", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("pgdown")
		if dv.scrollOffset < before {
			t.Errorf("pgdown should scroll down; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("b full-page-up", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("b")
		if dv.scrollOffset > before {
			t.Errorf("b should scroll up; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("ctrl+b full-page-up", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("ctrl+b")
		if dv.scrollOffset > before {
			t.Errorf("ctrl+b should scroll up; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("pgup full-page-up", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("pgup")
		if dv.scrollOffset > before {
			t.Errorf("pgup should scroll up; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("d half-page-down", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("d")
		if dv.scrollOffset < before {
			t.Errorf("d should scroll down; offset went from %d to %d", before, dv.scrollOffset)
		}
		_ = halfPage
	})

	t.Run("ctrl+d half-page-down", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("ctrl+d")
		if dv.scrollOffset < before {
			t.Errorf("ctrl+d should scroll down; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("u half-page-up", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("u")
		if dv.scrollOffset > before {
			t.Errorf("u should scroll up; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("ctrl+u half-page-up", func(t *testing.T) {
		dv := setup()
		before := dv.scrollOffset
		dv.HandleKey("ctrl+u")
		if dv.scrollOffset > before {
			t.Errorf("ctrl+u should scroll up; offset went from %d to %d", before, dv.scrollOffset)
		}
	})

	t.Run("g go-to-top", func(t *testing.T) {
		dv := setup()
		dv.HandleKey("g")
		if dv.scrollOffset != 0 {
			t.Errorf("g should scroll to top, got offset %d", dv.scrollOffset)
		}
	})

	t.Run("G go-to-bottom", func(t *testing.T) {
		dv := setup()
		dv.HandleKey("G")
		// After G, another ScrollDown should not change offset (already at bottom).
		after := dv.scrollOffset
		dv.ScrollDown(1)
		if dv.scrollOffset != after {
			t.Errorf("G should place view at bottom; subsequent scroll changed offset from %d to %d", after, dv.scrollOffset)
		}
	})
}
