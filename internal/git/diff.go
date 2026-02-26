package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotGitRepo is returned when the directory is not a git repository.
var ErrNotGitRepo = errors.New("not a git repository")

// FetchDiff returns the unified diff for the given directory.
//
// It combines two diffs:
//  1. Branch diff: commits on the current branch not on the base (origin/main,
//     origin/master, main, or master). Uses git diff <base>...HEAD.
//  2. Uncommitted diff: staged + unstaged changes against HEAD (git diff HEAD).
//
// When no base branch is found, only the uncommitted diff is returned.
//
// Returns ("", nil) for a clean tree.
// Returns ("", ErrNotGitRepo) if dir is not a git repo.
func FetchDiff(dir string) (string, error) {
	if !IsGitRepo(dir) {
		return "", ErrNotGitRepo
	}

	var parts []string

	// Part 1: branch-level diff against base
	base := findBase(dir)
	if base != "" {
		out, err := runDiff(dir, "diff", base+"...HEAD")
		if err != nil {
			return "", err
		}
		parts = append(parts, out)
	}

	// Part 2: uncommitted changes (staged + unstaged) vs HEAD
	out, err := runDiff(dir, "diff", "HEAD")
	if err != nil {
		return "", err
	}
	parts = append(parts, out)

	return strings.Join(parts, ""), nil
}

// runDiff executes git diff with the given args and returns the output.
// An exit code of 1 (differences found) is treated as success.
func runDiff(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// git diff exits 1 when differences are found — this is expected.
			// If stderr is non-empty it may indicate a reference error, but we
			// still return whatever output we got rather than failing the caller.
			return string(out), nil
		}
		return "", fmt.Errorf("git %v failed: %w", args, err)
	}
	return string(out), nil
}

// findBase returns the first known base ref that exists in the repo,
// or "" if none found.
func findBase(dir string) string {
	for _, base := range []string{"origin/main", "origin/master", "main", "master"} {
		cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", base)
		if err := cmd.Run(); err == nil {
			// Only use as base if we are NOT on that branch (triple-dot would be empty otherwise)
			// We still return it; the caller combines both diffs so it's fine either way.
			return base
		}
	}
	return ""
}

// DiffSummary parses a unified diff string and returns a short summary line.
// Returns "no changes" for an empty diff.
// Format: "N file(s), +X -Y"
func DiffSummary(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "no changes"
	}

	var files, additions, deletions int
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			files++
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			additions++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			deletions++
		}
	}

	noun := "file"
	if files != 1 {
		noun = "files"
	}
	return fmt.Sprintf("%d %s, +%d -%d", files, noun, additions, deletions)
}
