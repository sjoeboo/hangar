package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReviewDialog_InitiallyHidden(t *testing.T) {
	d := NewReviewDialog()
	if d.IsVisible() {
		t.Fatal("expected dialog to be hidden initially")
	}
}

func TestReviewDialog_ShowMakesVisible(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	if !d.IsVisible() {
		t.Fatal("expected dialog to be visible after Show")
	}
}

func TestReviewDialog_EscHidesDialog(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	action := d.HandleKey("esc")
	if action != "cancel" {
		t.Fatalf("expected action 'cancel', got %q", action)
	}
	if d.IsVisible() {
		t.Fatal("expected dialog to be hidden after esc")
	}
}

func TestReviewDialog_EnterWithEmptyInputDoesNothing(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	action := d.HandleKey("enter")
	if action != "" {
		t.Fatalf("expected empty action for enter with empty input, got %q", action)
	}
	if !d.IsVisible() {
		t.Fatal("expected dialog to remain visible")
	}
}

func TestReviewDialog_InputDetectsPRNumber(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	for _, r := range "42" {
		d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if !d.IsPRInput() {
		t.Fatal("expected '42' to be detected as a PR number")
	}
}

func TestReviewDialog_InputDetectsBranchName(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	for _, r := range "feature/my-branch" {
		d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if d.IsPRInput() {
		t.Fatal("expected 'feature/my-branch' to be detected as a branch, not a PR number")
	}
}

func TestReviewDialog_SetResolvedMovesToConfirmStep(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
	if d.step != 1 {
		t.Fatalf("expected step 1 after SetResolved, got %d", d.step)
	}
}

func TestReviewDialog_ConfirmStepEnterReturnsConfirm(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
	action := d.HandleKey("enter")
	if action != "confirm" {
		t.Fatalf("expected action 'confirm', got %q", action)
	}
}

func TestReviewDialog_ConfirmStepEscGoesBack(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")
	action := d.HandleKey("esc")
	if action != "" {
		t.Fatalf("expected empty action for esc in confirm step, got %q", action)
	}
	if d.step != 0 {
		t.Fatalf("expected step 0 after esc in confirm step, got %d", d.step)
	}
}

func TestReviewDialog_GetReviewValues(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	d.SetResolved("feature/auth-fix", "Fix auth bug", true, "42")

	branch, prNum, sessionName, initialPrompt := d.GetReviewValues()
	if branch != "feature/auth-fix" {
		t.Errorf("branch: want %q got %q", "feature/auth-fix", branch)
	}
	if prNum != "42" {
		t.Errorf("prNum: want %q got %q", "42", prNum)
	}
	if sessionName != "review/pr-42" {
		t.Errorf("sessionName: want %q got %q", "review/pr-42", sessionName)
	}
	if initialPrompt != "/pr-review 42" {
		t.Errorf("initialPrompt: want %q got %q", "/pr-review 42", initialPrompt)
	}
}

func TestReviewDialog_GetReviewValuesForBranch(t *testing.T) {
	d := NewReviewDialog()
	d.Show("hangar", "/home/user/code/hangar")
	d.SetSize(120, 40)
	d.SetResolved("feature/my-branch", "", false, "")

	branch, prNum, sessionName, initialPrompt := d.GetReviewValues()
	if branch != "feature/my-branch" {
		t.Errorf("branch: want %q got %q", "feature/my-branch", branch)
	}
	if prNum != "" {
		t.Errorf("prNum: want empty got %q", prNum)
	}
	if sessionName != "review/feature/my-branch" {
		t.Errorf("sessionName: want %q got %q", "review/feature/my-branch", sessionName)
	}
	if initialPrompt != "/pr-review feature/my-branch" {
		t.Errorf("initialPrompt: want %q got %q", "/pr-review feature/my-branch", initialPrompt)
	}
}
