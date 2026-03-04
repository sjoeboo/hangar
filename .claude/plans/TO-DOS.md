# Current Tasks — PR Management Overhaul

## In Progress
- [ ] Phase 3: PR detail overlay (internal/ui/pr_detail.go)
- [ ] Phase 6: Review session creation (git.go + 's' key handler)

## Completed This Session
- [x] Phase 1: internal/pr/ package (types.go, manager.go, fetch.go, actions.go)
- [x] Phase 2: TUI PR dashboard overhaul (home.go — prManager, tabs, new key actions)
- [x] Phase 4: API server (pr.Manager integrated, 5 new endpoints in pr_handlers.go)
- [x] Phase 5: WebUI (PROverview tabs, PRDetail/PRDiff/PRConversation components)

## Pending
- [ ] renderPROverview() tab bar visual (tabs in state but header not yet updated)
- [ ] Wire sendTextDialog PR comment submit (pendingPRComment set but not consumed)
- [ ] Phase 3: pr_detail.go full-screen overlay with Overview/Diff/Conversation tabs
- [ ] Phase 6: NormalizeRepoURL + 's' key review session creation
