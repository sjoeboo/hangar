# Current Tasks — PR Management

## Open

### TUI + WebUI
- [ ] **Author column**: Add `author` column to both TUI and WebUI PR tables
  - TUI (`home.go`): insert after Age in `renderPRRow`
  - WebUI (`PROverview.tsx`): add `<th>Author</th>` + `<td>{pr.author}</td>` after Age col

### WebUI Bugs
- [ ] **Approve submit does nothing**: POST `/api/v1/prs/review` fails silently
  - Check route registration in `server.go` ~line 108
  - Check `reviewMutation` error handling in `PRDetail.tsx` — add visible error
  - Check `isOwn` guard (Approve only shows when `!isOwn && isOpen`)
- [ ] **No approval column**: `review_decision` not showing for some PRs
  - Session PRs from `FetchSessionPR` include `reviewDecision` ✓
  - Global PRs from `enrichChecksForPRs` include it ✓
  - Check if enrichment is actually running in standalone mode (it runs inside `refreshMyPRs`/`refreshReviewPRs`)
- [ ] **Large PR diff crashes UI**: `PRDiff.tsx` renders all files/diff without virtualization
  - Truncate or lazy-expand file diffs; cap diff content size

## Completed This Session
- [x] PR overview — age, draft toggle/dimming, GHE host strip (WebUI)
- [x] Wire pr.Manager into standalone web server (was passing nil → 503)
- [x] Fix sidebar PR badge count (was counting sessions, now counts all PRs)
- [x] Fix JSON tags on PRDetail/Comment/Review/FileChange (Diff/Conversation tabs were empty)
- [x] Dual-host PR search — Mine/ReviewReq now searches github.com AND GHE
- [x] Column table layout in WebUI PR overview
- [x] Extract usePRDashboard hook (fixed make build-all TS6133 error)

## Previously Completed
- [x] Phase 1: internal/pr/ package
- [x] Phase 2: TUI PR dashboard (home.go)
- [x] Phase 3: PRDetailOverlay (pr_detail.go)
- [x] Phase 4: apiserver pr_handlers.go
- [x] Phase 5: WebUI PR components
- [x] Phase 6: RepoFromDir + 's' key review session
