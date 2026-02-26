package ui

import (
	"strings"
	"testing"
)

func TestPreview_DiffStat_Shown(t *testing.T) {
	p := NewPreview()
	p.SetSize(80, 24)
	p.SetContent("some content", "my-session")
	p.DiffStat = "3 files, +47 -12"

	out := p.View()
	if !strings.Contains(out, "3 files, +47 -12") {
		t.Errorf("expected diffstat in preview output, got:\n%s", out)
	}
}

func TestPreview_DiffStat_Empty_NotShown(t *testing.T) {
	p := NewPreview()
	p.SetSize(80, 24)
	p.SetContent("some content", "my-session")
	p.DiffStat = ""

	out := p.View()
	// When DiffStat is empty there should be no diffstat marker
	if strings.Contains(out, "~ ") {
		t.Errorf("expected no diffstat line in preview output when DiffStat is empty, got:\n%s", out)
	}
}
