package pr

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Approve submits an approving review for the PR.
// body may be empty.
func Approve(ghPath, repo string, number int, body string) error {
	args := []string{"pr", "review", itoa(number), "--repo", repoArg(repo), "--approve"}
	if body != "" {
		args = append(args, "-b", body)
	}
	return runGH(ghPath, repo, args)
}

// RequestChanges submits a "request changes" review for the PR.
// body is required by the GitHub API.
func RequestChanges(ghPath, repo string, number int, body string) error {
	if body == "" {
		body = "Changes requested."
	}
	args := []string{"pr", "review", itoa(number), "--repo", repoArg(repo), "--request-changes", "-b", body}
	return runGH(ghPath, repo, args)
}

// AddComment adds a PR-level comment. For inline comments (path != ""), a reply
// is posted to the relevant file thread. Note: the gh CLI does not support
// posting new inline comments via `gh pr comment`; inline comments require the
// GitHub API directly. For now this posts a PR-level comment with file context.
func AddComment(ghPath, repo string, number int, body, path string, line int) error {
	commentBody := body
	if path != "" && line > 0 {
		commentBody = fmt.Sprintf("**%s:%d**\n\n%s", path, line, body)
	}
	args := []string{"pr", "comment", itoa(number), "--repo", repoArg(repo), "--body", commentBody}
	return runGH(ghPath, repo, args)
}

// Close closes the PR.
func Close(ghPath, repo string, number int) error {
	args := []string{"pr", "close", itoa(number), "--repo", repoArg(repo)}
	return runGH(ghPath, repo, args)
}

// Reopen reopens a closed PR.
func Reopen(ghPath, repo string, number int) error {
	args := []string{"pr", "reopen", itoa(number), "--repo", repoArg(repo)}
	return runGH(ghPath, repo, args)
}

// ConvertToReady marks a draft PR as ready for review.
func ConvertToReady(ghPath, repo string, number int) error {
	args := []string{"pr", "ready", itoa(number), "--repo", repoArg(repo)}
	return runGH(ghPath, repo, args)
}

// ConvertToDraft converts a ready PR back to draft.
func ConvertToDraft(ghPath, repo string, number int) error {
	args := []string{"pr", "ready", itoa(number), "--repo", repoArg(repo), "--undo"}
	return runGH(ghPath, repo, args)
}

// runGH executes a gh command, setting GH_HOST if the repo is on a GHE instance.
func runGH(ghPath, repo string, args []string) error {
	cmd := exec.Command(ghPath, args...)
	if host := hostFromRepo(repo); host != "" && host != "github.com" {
		cmd.Env = append(os.Environ(), "GH_HOST="+host)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh %s: %w\n%s", strings.Join(args[:2], " "), err, string(out))
	}
	return nil
}

// repoArg returns the repo in the form the gh CLI expects (strips host prefix for GHE).
func repoArg(repo string) string {
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) == 3 {
		return parts[1] + "/" + parts[2]
	}
	return repo
}

// hostFromRepo extracts the hostname from repos stored as "host/owner/repo".
// Returns "" for standard "owner/repo" format (implies github.com).
func hostFromRepo(repo string) string {
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) == 3 {
		return parts[0]
	}
	return ""
}
