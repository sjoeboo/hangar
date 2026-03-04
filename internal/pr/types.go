// Package pr provides a unified PR data layer for Hangar.
// It consolidates PR fetching, caching, and actions that were previously
// split between the TUI (home.go) and the API server (apiserver/pr.go).
package pr

import "time"

// PRSource identifies how a PR was discovered.
type PRSource int

const (
	// SourceSession indicates the PR is linked to a Hangar worktree session.
	SourceSession PRSource = iota
	// SourceMine indicates the PR was authored by the current gh user.
	SourceMine
	// SourceReviewRequested indicates the current gh user was asked to review this PR.
	SourceReviewRequested
)

// PR holds pull-request metadata. It is the core data type for the PR dashboard.
type PR struct {
	Number        int
	Title         string
	Body          string
	State         string // OPEN, DRAFT, MERGED, CLOSED
	IsDraft       bool
	URL           string
	Repo          string // "owner/repo"
	HeadBranch    string
	BaseBranch    string
	Author        string
	ReviewDecision string // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, "" (empty = none)
	CommentCount  int

	ChecksPassed  int
	ChecksFailed  int
	ChecksPending int
	HasChecks     bool

	CreatedAt time.Time
	UpdatedAt time.Time

	Source    PRSource
	SessionID string // non-empty if linked to a Hangar session
}

// PRDetail extends PR with full detail loaded lazily (on demand).
type PRDetail struct {
	PR
	Mergeability string     `json:"mergeability,omitempty"`
	Comments     []Comment  `json:"comments"`
	Reviews      []Review   `json:"reviews"`
	Files        []FileChange `json:"files"`
	DiffContent  string     `json:"diff_content,omitempty"`
}

// Comment represents a PR-level or inline review comment.
type Comment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path,omitempty"` // empty = PR-level comment
	Line      int       `json:"line,omitempty"`
}

// Review represents a submitted review (approve, request-changes, comment).
type Review struct {
	Author    string    `json:"author"`
	State     string    `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Comments  []Comment `json:"comments"` // inline review comments
}

// FileChange describes a single file changed by the PR.
type FileChange struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"` // added, modified, deleted, renamed
}

// Key returns a unique cache key for this PR: "owner/repo#number".
func (p *PR) Key() string {
	return p.Repo + "#" + itoa(p.Number)
}

// NumberStr returns the PR number as a string (e.g. "42").
func NumberStr(n int) string { return itoa(n) }

// itoa is a minimal int-to-string helper to avoid importing strconv in types.go.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
