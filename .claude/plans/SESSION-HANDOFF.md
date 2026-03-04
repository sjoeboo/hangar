# Session Handoff — PR Management Overhaul

## What Was Accomplished

- **Phase 1** (`internal/pr/`): New unified PR package with `Manager`, `PR`, `PRDetail`, `Comment`, `Review`, `FileChange` types; `FetchMyPRs`, `FetchReviewRequestedPRs`, `FetchSessionPR`, `FetchDetail`; `Approve`, `RequestChanges`, `AddComment`, `Close`, `Reopen`, `ConvertToReady/Draft` actions
- **Phase 2** (home.go): `prManager *prpkg.Manager` field + `prViewTab int`; `prViewPRs()` with UICache fallback; `handlePRViewKey` updated with Tab/a/c/C/o/r actions; `prActionResultMsg`/`errMsg` types; `pendingPRComment` field; manager bridged to apiserver.New()
- **Phase 4** (apiserver): `internalPRCache` removed, `pr.Manager` dependency, `pr_handlers.go` with 5 endpoints, `PRFullInfo`/`PRDashboardResponse`/`ReviewActionRequest`/`CommentRequest`/`StateActionRequest` types
- **Phase 5** (webui): `PRFullInfo`, `PRDashboard`, `PRDetail`, `PRComment`, `PRReview`, `PRFileChange` TS types; `getPRDashboard/getPRDetail/submitPRReview/addPRComment/changePRState` API functions; PROverview filter tabs; `PRDetail.tsx`, `PRDiff.tsx`, `PRConversation.tsx` components

## Current State

- Branch: `pr-mgmt` (worktree at `.worktrees/pr-mgmt`)
- Tests: `go test ./internal/pr/ ./internal/apiserver/ ./internal/ui/` — all pass
- Build: `go build ./internal/pr/ ./internal/apiserver/ ./internal/ui/` — clean
- Pre-existing failure: `internal/webui` needs `npm run build` first (not our problem)

## Two Remaining Gaps Before Phase 2 is Visually Complete

1. **renderPROverview() tab bar** — `prViewTab` state exists but `renderPROverview()` still shows old header without tab bar. Add tab bar rows (line ~6570 in home.go): `[All (N)] [Mine (N)] [Review Requests (N)] [Sessions (N)]`
2. **sendTextDialog PR comment flow** — `pendingPRComment *prpkg.PR` field is set when `c` pressed, but `sendTextDialog` submit handler doesn't check it. Find the `sendTextResultMsg` handler (~line 2563 home.go) or wherever sendTextDialog submission dispatches the text, and add: if `h.pendingPRComment != nil`, fire `prpkg.AddComment(...)` and clear field.

## Open Issues / Next Steps (in order)

1. **Fix renderPROverview tab bar** — visual only, ~10 lines
2. **Fix sendTextDialog PR comment** — wire `pendingPRComment` in the submit path
3. **Phase 3: Create `internal/ui/pr_detail.go`**
   - `PRDetailOverlay` struct with `visible bool`, `pr *prpkg.PR`, `detail *prpkg.PRDetail`, `tab int` (0=Overview,1=Diff,2=Conversation), `scrollOffset int`, `loading bool`
   - Follow `DiffView` pattern (`internal/ui/diff_view.go`) for struct layout
   - Wire into home.go in 7 places: struct field, init, key routing (Enter in PR view opens it), mouse guard, trigger, SetSize, View check
   - New message types: `prDetailLoadingMsg{prKey string}`, `prDetailLoadedMsg{detail *prpkg.PRDetail}`
4. **Phase 6: `internal/git/git.go`** — add `NormalizeRepoURL(url string) string`
   - Strips `git@`, `https://`, `.git`; lowercases → returns `host/owner/repo`
5. **Phase 6: `s` key in handlePRViewKey** — scan `h.projects` for match, call handleAdd with branch=pr.HeadBranch, after `worktreeCreatedForNewSessionMsg` auto-send review prompt

## Important Context

- Import alias for pr package in home.go and update_handlers.go: `prpkg "github.com/sjoeboo/hangar/internal/pr"`
- `prCacheEntry` (home.go ~line 487) is KEPT for backward compat with UICache and worktreeFinishDialog.SetPR()
- `apiserver.New()` signature is now: `New(cfg, watcher, getInstances, getPRInfo, triggerReload, prManager, profile, version)`
- `prpkg.NumberStr(n int) string` helper exists in types.go
- `sendTextDialog.Show(sessionTitle string)` — only takes a title, no callback
- `setStatusMessage` does NOT exist — use `h.setError(fmt.Errorf(...))` for transient status

## Commands to Run First

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/pr-mgmt
go build ./internal/pr/ ./internal/apiserver/ ./internal/ui/  # verify clean
go test ./internal/pr/ ./internal/apiserver/ ./internal/ui/   # verify tests pass
```
