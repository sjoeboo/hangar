package pr

import (
	"context"
	"log/slog"
	"os/exec"
	"sort"
	"sync"
	"time"
)

const (
	// globalRefreshInterval controls how often Mine/ReviewRequested lists are refreshed.
	globalRefreshInterval = 5 * time.Minute
	// sessionPRTTL controls per-session PR cache staleness.
	sessionPRTTL = 60 * time.Second
	// detailTTL controls how long a fetched PRDetail stays cached.
	detailTTL = 5 * time.Minute
)

// Manager is the central PR data layer. It maintains three PR lists:
//   - sessionPRs: per-session, updated externally via UpdateSessionPR
//   - myPRs: PRs authored by the current gh user
//   - reviewPRs: PRs where the current user has been asked to review
//
// Background goroutines refresh myPRs and reviewPRs every 5 minutes.
// All public methods are safe for concurrent use.
type Manager struct {
	mu     sync.RWMutex
	ghPath string
	ghUser string

	// per-session PR state (keyed by sessionID)
	sessionPRs    map[string]*PR
	sessionFetched map[string]time.Time

	// global lists
	myPRs              []*PR
	reviewPRs          []*PR
	myPRsLastFetch     time.Time
	reviewPRsLastFetch time.Time

	// detail cache (keyed by "owner/repo#number")
	detailCache     map[string]*PRDetail
	detailFetchedAt map[string]time.Time

	// change notifications
	onChangeMu sync.Mutex
	onChange   []func()

	// refreshCh is a buffered channel used by TriggerRefresh to schedule a
	// one-shot re-fetch of myPRs/reviewPRs. Buffer of 1 coalesces duplicates.
	refreshCh chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Manager. Call Start() to begin background refresh loops.
func New() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		sessionPRs:      make(map[string]*PR),
		sessionFetched:  make(map[string]time.Time),
		detailCache:     make(map[string]*PRDetail),
		detailFetchedAt: make(map[string]time.Time),
		refreshCh:       make(chan struct{}, 1),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// TriggerRefresh schedules an immediate re-fetch of the Mine and ReviewRequested lists.
// If a refresh is already pending, this is a no-op. Safe to call from any goroutine.
func (m *Manager) TriggerRefresh() {
	select {
	case m.refreshCh <- struct{}{}:
	default: // already queued
	}
}

// Start initialises gh detection and begins background refresh loops.
// It is non-blocking; the loops run as goroutines.
func (m *Manager) Start() {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		slog.Warn("pr_manager: gh not found; PR features disabled")
		return
	}
	m.mu.Lock()
	m.ghPath = ghPath
	m.mu.Unlock()

	// Detect gh user asynchronously to avoid blocking startup
	go func() {
		user := DetectGHUser(ghPath)
		m.mu.Lock()
		m.ghUser = user
		m.mu.Unlock()
		slog.Debug("pr_manager: gh user detected", "user", user)

		if user == "" {
			slog.Warn("pr_manager: gh user empty; skipping global PR refresh")
			return
		}

		// Initial fetch of global lists
		m.refreshMyPRs()
		m.refreshReviewPRs()

		// Periodic refresh
		ticker := time.NewTicker(globalRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.refreshMyPRs()
				m.refreshReviewPRs()
			case <-m.refreshCh:
				m.refreshMyPRs()
				m.refreshReviewPRs()
			}
		}
	}()
}

// Stop halts all background goroutines.
func (m *Manager) Stop() {
	m.cancel()
}

// GHPath returns the resolved path to the gh binary (empty if not found).
func (m *Manager) GHPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ghPath
}

// GHUser returns the detected GitHub username (empty until Start() completes the detection).
func (m *Manager) GHUser() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ghUser
}

// RegisterOnChange registers a callback invoked whenever PR data changes.
// Callbacks are called with no lock held and must not call back into Manager.
func (m *Manager) RegisterOnChange(fn func()) {
	m.onChangeMu.Lock()
	defer m.onChangeMu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// notifyChange invokes all registered onChange callbacks.
func (m *Manager) notifyChange() {
	m.onChangeMu.Lock()
	cbs := make([]func(), len(m.onChange))
	copy(cbs, m.onChange)
	m.onChangeMu.Unlock()
	for _, fn := range cbs {
		fn()
	}
}

// GetAll returns a merged, deduplicated list of all known PRs, sorted by
// UpdatedAt descending. Session PRs and global PRs are merged; duplicate
// URLs are collapsed, preferring the session-linked entry.
func (m *Manager) GetAll() []*PR {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]*PR) // keyed by URL

	// Session PRs take precedence (they carry SessionID)
	for _, p := range m.sessionPRs {
		if p != nil {
			seen[p.URL] = p
		}
	}
	// Merge global lists (don't overwrite session PRs)
	for _, list := range [][]*PR{m.myPRs, m.reviewPRs} {
		for _, p := range list {
			if _, exists := seen[p.URL]; !exists {
				seen[p.URL] = p
			}
		}
	}

	out := make([]*PR, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

// GetMine returns PRs authored by the current user.
func (m *Manager) GetMine() []*PR {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*PR, len(m.myPRs))
	copy(out, m.myPRs)
	return out
}

// GetReviewRequested returns PRs where review was requested from the current user.
func (m *Manager) GetReviewRequested() []*PR {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*PR, len(m.reviewPRs))
	copy(out, m.reviewPRs)
	return out
}

// GetSessionPRs returns a snapshot of the per-session PR map.
func (m *Manager) GetSessionPRs() map[string]*PR {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*PR, len(m.sessionPRs))
	for k, v := range m.sessionPRs {
		out[k] = v
	}
	return out
}

// GetSessionPR returns the cached PR for a single session, plus whether a
// (possibly nil) result has been stored at all.
func (m *Manager) GetSessionPR(sessionID string) (pr *PR, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.sessionPRs[sessionID]
	return p, ok
}

// SessionPRStaleAt returns when the session PR entry was last fetched.
// exists is false if nothing has been cached yet.
func (m *Manager) SessionPRStaleAt(sessionID string) (fetchedAt time.Time, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.sessionFetched[sessionID]
	return t, ok
}

// SetSessionPR stores a (possibly nil) PR for a session directly.
// Used when the caller has already done the gh fetch (e.g. TUI migration path).
func (m *Manager) SetSessionPR(sessionID string, p *PR) {
	m.mu.Lock()
	m.sessionPRs[sessionID] = p
	m.sessionFetched[sessionID] = time.Now()
	m.mu.Unlock()
	m.notifyChange()
}

// UpdateSessionPR triggers an async fetch for the given worktree session.
// If a fresh result is already cached (within sessionPRTTL), this is a no-op.
func (m *Manager) UpdateSessionPR(sessionID, worktreePath string) {
	ghPath := m.GHPath()
	if ghPath == "" {
		return
	}

	m.mu.RLock()
	fetchedAt, hasFetch := m.sessionFetched[sessionID]
	m.mu.RUnlock()

	if hasFetch && time.Since(fetchedAt) < sessionPRTTL {
		return // still fresh
	}

	go func() {
		p, err := FetchSessionPR(ghPath, worktreePath, sessionID)
		if err != nil {
			slog.Debug("pr_manager: session PR fetch error", "session", sessionID, "err", err)
			return
		}
		m.mu.Lock()
		m.sessionPRs[sessionID] = p
		m.sessionFetched[sessionID] = time.Now()
		m.mu.Unlock()
		m.notifyChange()
	}()
}

// FetchDetail returns full PR detail. It is fetched lazily and cached for detailTTL.
// This call blocks until the fetch completes.
func (m *Manager) FetchDetail(repo string, number int) (*PRDetail, error) {
	key := repo + "#" + itoa(number)

	m.mu.RLock()
	detail, ok := m.detailCache[key]
	fetchedAt := m.detailFetchedAt[key]
	m.mu.RUnlock()

	if ok && time.Since(fetchedAt) < detailTTL {
		return detail, nil
	}

	ghPath := m.GHPath()
	if ghPath == "" {
		return nil, nil
	}

	d, err := FetchDetail(ghPath, repo, number)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.detailCache[key] = d
	m.detailFetchedAt[key] = time.Now()
	m.mu.Unlock()

	return d, nil
}

// InvalidateDetail removes a cached PRDetail so the next call to FetchDetail
// re-fetches from GitHub (e.g. after posting a review).
func (m *Manager) InvalidateDetail(repo string, number int) {
	key := repo + "#" + itoa(number)
	m.mu.Lock()
	delete(m.detailCache, key)
	delete(m.detailFetchedAt, key)
	m.mu.Unlock()
}

// inferGHHost returns the dominant GH host from known session PRs, or "".
// Used to set GH_HOST when running gh search prs on GHE instances.
func (m *Manager) inferGHHost() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.sessionPRs {
		if p != nil {
			if h := hostFromRepo(p.Repo); h != "" && h != "github.com" {
				return h
			}
		}
	}
	return ""
}

// refreshMyPRs fetches the current user's open PRs and updates the cache.
func (m *Manager) refreshMyPRs() {
	ghPath := m.GHPath()
	if ghPath == "" {
		return
	}
	prs, err := FetchMyPRs(ghPath, m.inferGHHost())
	if err != nil {
		slog.Debug("pr_manager: fetch my PRs error", "err", err)
		return
	}
	for _, p := range prs {
		p.Source = SourceMine
	}
	enrichChecksForPRs(ghPath, prs)
	m.mu.Lock()
	m.myPRs = prs
	m.myPRsLastFetch = time.Now()
	m.mu.Unlock()
	slog.Debug("pr_manager: my PRs refreshed", "count", len(prs))
	m.notifyChange()
}

// refreshReviewPRs fetches PRs where review was requested from the current user.
func (m *Manager) refreshReviewPRs() {
	ghPath := m.GHPath()
	if ghPath == "" {
		return
	}
	prs, err := FetchReviewRequestedPRs(ghPath, m.inferGHHost())
	if err != nil {
		slog.Debug("pr_manager: fetch review PRs error", "err", err)
		return
	}
	for _, p := range prs {
		p.Source = SourceReviewRequested
	}
	enrichChecksForPRs(ghPath, prs)
	m.mu.Lock()
	m.reviewPRs = prs
	m.reviewPRsLastFetch = time.Now()
	m.mu.Unlock()
	slog.Debug("pr_manager: review PRs refreshed", "count", len(prs))
	m.notifyChange()
}
