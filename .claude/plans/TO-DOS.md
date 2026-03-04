# Current Tasks — PR Management

## Next Up
- [ ] WebUI: Add same PR features as TUI (see SESSION-HANDOFF.md for full list)
  - [ ] Repo column (owner/repo, strip GHE host)
  - [ ] Age column (compact: <1d/3d/2w/14mo/2y from CreatedAt)
  - [ ] Draft dimming + hide/show toggle button
  - [ ] Review decision icon (✓ APPROVED, △ CHANGES_REQUESTED)
  - [ ] Comment input in PRDetail/PRConversation
  - [ ] Open in browser button in PRDetail header
  - [ ] Verify check status shows correctly (enriched data via enrichChecksForPRs)

## Completed This Session
- [x] Bug 2: Repo+HeadBranch in UICache fallback (prViewPRs)
- [x] Bug 1: Bridge prFetchedMsg → prManager.SetSessionPR with Repo/HeadBranch
- [x] Bug 3: GHE host inference (inferGHHost) for global PR search
- [x] Root fix: remoteURLToRepo preserves GHE host ("host/owner/repo")
- [x] Mine/ReviewReq empty: removed unsupported ghSearchFields (reviewDecision, statusCheckRollup, comments)
- [x] Draft state: fetchPRInfo now requests isDraft + uses StateFromSearchResult
- [x] TriggerRefresh() on Manager (buffered channel, called after first session PR)
- [x] Comment + browser open from PRDetailOverlay (c/o keys)
- [x] Repo column in TUI PR list (28 chars, strips GHE host)
- [x] Age column in TUI PR list (compact format, prAgeStr helper)
- [x] --archived=false filter in gh search prs
- [x] Draft dimming (ColorTextDim italic) + D toggle + hint bar label
- [x] Review decision icon column (✓/△, prApproved map + ReviewDecision from enrichment)
- [x] enrichChecksForPRs now also fetches reviewDecision

## Previously Completed
- [x] Phase 1: internal/pr/ package
- [x] Phase 2: TUI PR dashboard (home.go)
- [x] Phase 3: PRDetailOverlay (pr_detail.go)
- [x] Phase 4: apiserver pr_handlers.go
- [x] Phase 5: WebUI PR components
- [x] Phase 6: RepoFromDir + 's' key review session
