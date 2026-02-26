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
