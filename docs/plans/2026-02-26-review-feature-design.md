# Review Feature Design

**Date**: 2026-02-26
**Branch**: review-feature
**Status**: Approved

## Overview

Add a Claude-assisted PR review workflow to Hangar. From any project in the sidebar, press `v` to open a focused "Review PR" dialog. Hangar fetches the branch, creates a worktree, launches Claude, and delivers a `/pr-review` prompt automatically.

## Trigger

- Key: `v` (currently unbound)
- Context: works when a project group header **or** a session within a project is selected in the sidebar
- In both cases, Hangar derives `BaseDir` and `BaseBranch` from context

## Dialog Flow (`ReviewDialog`)

**New file**: `internal/ui/review_dialog.go`

### Step 1 — Input

```
╭─ Review PR ────────────────────────────────╮
│                                            │
│  Project: hangar                           │
│                                            │
│  PR # or branch  [                      ]  │
│                                            │
│  enter to continue · esc to cancel         │
╰────────────────────────────────────────────╯
```

User types a PR number (digits) or branch name and presses Enter.

### Step 2 — Resolve & Confirm

**If digits**: async `gh pr view <N> --json headRefName,title` resolves to branch + title:

```
╭─ Review PR ────────────────────────────────╮
│                                            │
│  Project: hangar                           │
│  PR #42: Fix authentication bug            │
│  Branch: feature/auth-fix                  │
│  Session: review/pr-42                     │
│                                            │
│  enter to create · esc to cancel           │
╰────────────────────────────────────────────╯
```

**If branch name**: skip resolution, show confirm view directly with session name `review/<branch>`.

**If `gh` lookup fails**: inline error shown, input stays editable. User can correct or proceed — failure falls back to treating input as a branch name.

## Session & Worktree Creation

Async operation on confirm (mirrors existing `worktreeCreatedForNewSessionMsg` flow):

1. **`git.FetchBranch(repoDir, branch)`** — new helper, runs `git fetch origin <branch>:<branch>` to pull remote branch locally
2. **`git.CreateWorktree(repoRoot, worktreePath, branchName)`** — existing function, no changes needed
   - Worktree location: `.worktrees/review-pr-<N>` or `.worktrees/review-<branch>` (subdirectory strategy)
3. **Session start** — existing machinery, working directory = worktree path, command = `claude`
4. **Prompt delivery** — after Claude reaches ready state, send `/pr-review <N>` or `/pr-review <branch>` as tmux keystrokes (reuses delayed-send mechanism from the `x` send-message flow)

## Session Naming

| Input | Session name | Worktree path |
|-------|-------------|---------------|
| PR `42` | `review/pr-42` | `.worktrees/review-pr-42` |
| Branch `feature/auth-fix` | `review/feature/auth-fix` | `.worktrees/review-feature-auth-fix` |

The `review/` prefix causes the session to group naturally in the sidebar with the project.

## Wiring in `home.go`

Standard 7-point dialog wiring:

1. Struct field: `reviewDialog *ReviewDialog`
2. Init: `h.reviewDialog = NewReviewDialog()`
3. Key routing: `case "v"` → `h.reviewDialog.Show(project)` when sidebar has project context
4. Mouse guard: block pass-through when dialog is visible
5. `SetSize` on window resize
6. `View()`: overlay when `h.reviewDialog.IsVisible()`
7. Result message handler for `reviewDialogResultMsg`

## New Message Types

Following existing patterns in `home.go`:

- `reviewPRResolvedMsg` — carries resolved branch name + PR title after `gh` lookup
- `reviewWorktreeCreatedMsg` — carries session info after worktree creation (mirrors `worktreeCreatedForNewSessionMsg`)

## Error Handling

| Failure | Behavior |
|---------|----------|
| `gh` lookup fails | Inline error in dialog; input stays editable; user can proceed treating input as branch name |
| `git fetch` fails | Status bar error (auto-dismiss); dialog closes |
| Branch already has worktree | Inline error: "Worktree already exists for this branch" |
| Not a git repo | `v` key is a no-op (same guard as worktree operations) |

## Cleanup

Review sessions are regular worktree sessions. Cleanup uses the existing `W` (worktree finish) flow — no new persistent state or cleanup logic needed.

## Files Changed

| File | Change |
|------|--------|
| `internal/ui/review_dialog.go` | New — `ReviewDialog` struct and dialog logic |
| `internal/git/git.go` | New `FetchBranch()` helper |
| `internal/ui/home.go` | 7-point dialog wiring + `v` key binding |
