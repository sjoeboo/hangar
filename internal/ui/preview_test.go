package ui

import (
	"strings"
	"testing"

	"github.com/sjoeboo/hangar/internal/tmux"
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

func TestPreview_DiffStatLine(t *testing.T) {
	p := NewPreview()

	// Empty DiffStat returns empty string
	if got := p.DiffStatLine(); got != "" {
		t.Errorf("expected empty string for empty DiffStat, got %q", got)
	}

	// Non-empty DiffStat returns a styled line containing the diffstat text
	p.DiffStat = "3 files, +47 -12"
	got := p.DiffStatLine()
	if got == "" {
		t.Error("expected non-empty string for non-empty DiffStat")
	}
	stripped := tmux.StripANSI(got)
	if !strings.Contains(stripped, "3 files, +47 -12") {
		t.Errorf("DiffStatLine() output %q does not contain diffstat", stripped)
	}
	if !strings.Contains(stripped, "~ ") {
		t.Errorf("DiffStatLine() output %q does not contain '~ ' prefix", stripped)
	}
}
