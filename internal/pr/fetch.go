package pr

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ghSearchFields are the JSON fields requested from `gh search prs`.
const ghSearchFields = "number,title,state,url,repository,author,isDraft,reviewDecision,statusCheckRollup,comments,createdAt,updatedAt"

// ghCheck is the common status-check shape returned by the gh API.
type ghCheck struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// ghViewFields are the JSON fields requested from `gh pr view` for a specific PR.
const ghViewFields = "number,title,body,state,url,author,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup,comments,reviews,files,headRefName,baseRefName,createdAt,updatedAt"

// DetectGHUser runs `gh api user` to find the authenticated username.
// Returns "" if gh is unavailable or unauthenticated.
func DetectGHUser(ghPath string) string {
	cmd := exec.Command(ghPath, "api", "user", "--jq", ".login")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// FetchMyPRs fetches open PRs authored by the current gh user.
func FetchMyPRs(ghPath string) ([]*PR, error) {
	return fetchSearchPRs(ghPath, "--author", "@me")
}

// FetchReviewRequestedPRs fetches open PRs where review has been requested from the current user.
func FetchReviewRequestedPRs(ghPath string) ([]*PR, error) {
	return fetchSearchPRs(ghPath, "--review-requested", "@me")
}

// fetchSearchPRs runs `gh search prs` with the given filter flag and parses the results.
func fetchSearchPRs(ghPath, filterFlag, filterValue string) ([]*PR, error) {
	cmd := exec.Command(ghPath, "search", "prs",
		filterFlag, filterValue,
		"--state", "open",
		"--limit", "50",
		"--json", ghSearchFields,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	type ghRepo struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	type ghAuthor struct {
		Login string `json:"login"`
	}
	type ghComments struct {
		TotalCount int `json:"totalCount"`
	}
	type ghSearchResult struct {
		Number         int        `json:"number"`
		Title          string     `json:"title"`
		State          string     `json:"state"`
		URL            string     `json:"url"`
		Repository     ghRepo     `json:"repository"`
		Author         ghAuthor   `json:"author"`
		IsDraft        bool       `json:"isDraft"`
		ReviewDecision string     `json:"reviewDecision"`
		StatusCheckRollup []ghCheck `json:"statusCheckRollup"`
		Comments       ghComments `json:"comments"`
		CreatedAt      time.Time  `json:"createdAt"`
		UpdatedAt      time.Time  `json:"updatedAt"`
	}

	var results []ghSearchResult
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, err
	}

	prs := make([]*PR, 0, len(results))
	for _, r := range results {
		p := &PR{
			Number:         r.Number,
			Title:          r.Title,
			State:          stateFromSearchResult(r.State, r.IsDraft),
			IsDraft:        r.IsDraft,
			URL:            r.URL,
			Repo:           r.Repository.NameWithOwner,
			Author:         r.Author.Login,
			ReviewDecision: r.ReviewDecision,
			CommentCount:   r.Comments.TotalCount,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		}
		parseChecks(p, r.StatusCheckRollup)
		prs = append(prs, p)
	}
	return prs, nil
}

// FetchSessionPR fetches PR info for a single worktree session directory.
// Returns nil (no error) if there is no PR for the current branch.
func FetchSessionPR(ghPath, worktreePath, sessionID string) (*PR, error) {
	type ghPR struct {
		Number            int        `json:"number"`
		Title             string     `json:"title"`
		State             string     `json:"state"`
		URL               string     `json:"url"`
		IsDraft           bool       `json:"isDraft"`
		ReviewDecision    string     `json:"reviewDecision"`
		HeadRefName       string     `json:"headRefName"`
		BaseRefName       string     `json:"baseRefName"`
		StatusCheckRollup []ghCheck  `json:"statusCheckRollup"`
		CreatedAt         time.Time  `json:"createdAt"`
		UpdatedAt         time.Time  `json:"updatedAt"`
	}

	cmd := exec.Command(ghPath, "pr", "view",
		"--json", "number,title,state,url,isDraft,reviewDecision,headRefName,baseRefName,statusCheckRollup,createdAt,updatedAt",
	)
	cmd.Dir = worktreePath
	if host := ghHostFromDir(worktreePath); host != "" && host != "github.com" {
		cmd.Env = append(os.Environ(), "GH_HOST="+host)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, nil // no PR found (expected case)
	}

	var raw ghPR
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	// Infer repo from remote URL
	repo := repoFromDir(worktreePath)

	p := &PR{
		Number:         raw.Number,
		Title:          raw.Title,
		State:          stateFromSearchResult(raw.State, raw.IsDraft),
		IsDraft:        raw.IsDraft,
		URL:            raw.URL,
		Repo:           repo,
		HeadBranch:     raw.HeadRefName,
		BaseBranch:     raw.BaseRefName,
		ReviewDecision: raw.ReviewDecision,
		CreatedAt:      raw.CreatedAt,
		UpdatedAt:      raw.UpdatedAt,
		Source:         SourceSession,
		SessionID:      sessionID,
	}
	parseChecks(p, raw.StatusCheckRollup)
	return p, nil
}

// FetchDetail fetches full PR detail including diff, comments, and reviews.
func FetchDetail(ghPath, repo string, number int) (*PRDetail, error) {
	type ghAuthor struct {
		Login string `json:"login"`
	}
	type ghCommentNode struct {
		DatabaseID int64     `json:"databaseId"`
		Author     ghAuthor  `json:"author"`
		Body       string    `json:"body"`
		CreatedAt  time.Time `json:"createdAt"`
		Path       string    `json:"path"`
		Line       int       `json:"line"`
	}
	type ghReviewComment struct {
		DatabaseID int64     `json:"databaseId"`
		Author     ghAuthor  `json:"author"`
		Body       string    `json:"body"`
		CreatedAt  time.Time `json:"createdAt"`
		Path       string    `json:"path"`
		Line       int       `json:"line"`
	}
	type ghReview struct {
		Author    ghAuthor          `json:"author"`
		State     string            `json:"state"`
		Body      string            `json:"body"`
		CreatedAt time.Time         `json:"createdAt"`
		Comments  []ghReviewComment `json:"comments"`
	}
	type ghFile struct {
		Path      string `json:"path"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Status    string `json:"changeType"`
	}
	type ghPRDetail struct {
		Number            int             `json:"number"`
		Title             string          `json:"title"`
		Body              string          `json:"body"`
		State             string          `json:"state"`
		URL               string          `json:"url"`
		Author            ghAuthor        `json:"author"`
		IsDraft           bool            `json:"isDraft"`
		MergeStateStatus  string          `json:"mergeStateStatus"`
		ReviewDecision    string          `json:"reviewDecision"`
		HeadRefName       string          `json:"headRefName"`
		BaseRefName       string          `json:"baseRefName"`
		StatusCheckRollup []ghCheck       `json:"statusCheckRollup"`
		Comments          []ghCommentNode `json:"comments"`
		Reviews           []ghReview      `json:"reviews"`
		Files             []ghFile        `json:"files"`
		CreatedAt         time.Time       `json:"createdAt"`
		UpdatedAt         time.Time       `json:"updatedAt"`
	}

	// Determine GH_HOST from repo if it's a GHE instance
	var env []string
	parts := strings.SplitN(repo, "/", 3) // handle "host/owner/repo" for GHE
	ghHost := ""
	ghRepo := repo
	if len(parts) == 3 {
		ghHost = parts[0]
		ghRepo = parts[1] + "/" + parts[2]
	}

	cmd := exec.Command(ghPath, "pr", "view", itoa(number),
		"--repo", ghRepo,
		"--json", "number,title,body,state,url,author,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup,comments,reviews,files,headRefName,baseRefName,createdAt,updatedAt",
	)
	if ghHost != "" && ghHost != "github.com" {
		env = append(os.Environ(), "GH_HOST="+ghHost)
		cmd.Env = env
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var raw ghPRDetail
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	p := &PR{
		Number:         raw.Number,
		Title:          raw.Title,
		Body:           raw.Body,
		State:          stateFromSearchResult(raw.State, raw.IsDraft),
		IsDraft:        raw.IsDraft,
		URL:            raw.URL,
		Repo:           repo,
		HeadBranch:     raw.HeadRefName,
		BaseBranch:     raw.BaseRefName,
		Author:         raw.Author.Login,
		ReviewDecision: raw.ReviewDecision,
		CreatedAt:      raw.CreatedAt,
		UpdatedAt:      raw.UpdatedAt,
	}
	for _, c := range raw.StatusCheckRollup {
		p.HasChecks = true
		switch c.Status {
		case "COMPLETED":
			switch c.Conclusion {
			case "SUCCESS", "SKIPPED", "NEUTRAL":
				p.ChecksPassed++
			default:
				p.ChecksFailed++
			}
		default:
			p.ChecksPending++
		}
	}

	detail := &PRDetail{PR: *p}

	switch raw.MergeStateStatus {
	case "CLEAN":
		detail.Mergeability = "MERGEABLE"
	case "DIRTY":
		detail.Mergeability = "CONFLICTING"
	default:
		detail.Mergeability = "UNKNOWN"
	}

	for _, c := range raw.Comments {
		detail.Comments = append(detail.Comments, Comment{
			ID:        c.DatabaseID,
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
			Path:      c.Path,
			Line:      c.Line,
		})
	}

	for _, r := range raw.Reviews {
		rev := Review{
			Author:    r.Author.Login,
			State:     r.State,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
		}
		for _, c := range r.Comments {
			rev.Comments = append(rev.Comments, Comment{
				ID:        c.DatabaseID,
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
				Path:      c.Path,
				Line:      c.Line,
			})
		}
		detail.Reviews = append(detail.Reviews, rev)
	}

	for _, f := range raw.Files {
		detail.Files = append(detail.Files, FileChange{
			Path:      f.Path,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Status:    f.Status,
		})
	}

	// Fetch diff separately
	diffCmd := exec.Command(ghPath, "pr", "diff", itoa(number), "--repo", ghRepo)
	if ghHost != "" && ghHost != "github.com" {
		diffCmd.Env = env
	}
	if diffOut, err := diffCmd.Output(); err == nil {
		detail.DiffContent = string(diffOut)
	}

	return detail, nil
}

// parseChecks parses the gh statusCheckRollup structure into a PR.
func parseChecks(p *PR, checks []ghCheck) {
	p.HasChecks = len(checks) > 0
	for _, c := range checks {
		switch c.Status {
		case "COMPLETED":
			switch c.Conclusion {
			case "SUCCESS", "SKIPPED", "NEUTRAL":
				p.ChecksPassed++
			default:
				p.ChecksFailed++
			}
		default:
			p.ChecksPending++
		}
	}
}

// stateFromSearchResult normalises the gh API state value, treating draft
// as a separate state (GitHub API returns "OPEN" for drafts).
func stateFromSearchResult(state string, isDraft bool) string {
	if isDraft && state == "OPEN" {
		return "DRAFT"
	}
	return state
}

// repoFromDir infers "owner/repo" from the git remote URL of a directory.
func repoFromDir(dir string) string {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return remoteURLToRepo(strings.TrimSpace(string(out)))
}

// remoteURLToRepo converts a git remote URL to "owner/repo" format.
//
//	git@github.com:owner/repo.git  → "owner/repo"
//	https://github.com/owner/repo  → "owner/repo"
func remoteURLToRepo(u string) string {
	// git@HOST:owner/repo.git
	if after, ok := strings.CutPrefix(u, "git@"); ok {
		_, path, _ := strings.Cut(after, ":")
		return strings.TrimSuffix(path, ".git")
	}
	// https://HOST/owner/repo[.git]
	for _, scheme := range []string{"https://", "http://"} {
		if after, ok := strings.CutPrefix(u, scheme); ok {
			// after = "github.com/owner/repo.git"
			_, path, found := strings.Cut(after, "/")
			if found {
				return strings.TrimSuffix(path, ".git")
			}
		}
	}
	return ""
}

// extractRemoteHost parses a git remote URL and returns just the hostname.
func extractRemoteHost(u string) string {
	if after, ok := strings.CutPrefix(u, "git@"); ok {
		host, _, _ := strings.Cut(after, ":")
		return host
	}
	for _, scheme := range []string{"https://", "http://"} {
		if after, ok := strings.CutPrefix(u, scheme); ok {
			host, _, _ := strings.Cut(after, "/")
			return host
		}
	}
	return ""
}

// ghHostFromDir returns the git remote hostname for use as GH_HOST.
func ghHostFromDir(dir string) string {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return extractRemoteHost(strings.TrimSpace(string(out)))
}
