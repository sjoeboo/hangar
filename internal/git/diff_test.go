package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ghe.spotify.net/mnicholson/hangar/internal/git"
)

// makeRepo creates a temporary git repo for testing
func makeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--initial-branch=main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial")
	return dir
}

func TestFetchDiff_CleanTree(t *testing.T) {
	dir := makeRepo(t)
	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for clean tree, got: %q", diff)
	}
}

func TestFetchDiff_WithChanges(t *testing.T) {
	dir := makeRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "world") {
		t.Errorf("expected diff to contain 'world', got: %q", diff)
	}
	if !strings.Contains(diff, "hello.txt") {
		t.Errorf("expected diff to mention hello.txt, got: %q", diff)
	}
}

func TestFetchDiff_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := git.FetchDiff(dir)
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestFetchDiff_WorktreeBranch(t *testing.T) {
	dir := makeRepo(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("checkout", "-b", "feature/test")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "add new.txt")

	diff, err := git.FetchDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "new.txt") {
		t.Errorf("expected diff to contain new.txt, got: %q", diff)
	}
}
