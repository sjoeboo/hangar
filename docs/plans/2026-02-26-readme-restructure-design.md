# README Restructure + Screenshots Design

**Date:** 2026-02-26
**Status:** Approved

## Goal

Split the monolithic README into focused docs, add screenshots to make features immediately visible, and give the README a clear landing-page feel.

## File Structure

```
README.md                          ← landing page (new, shorter)
assets/
  screenshots/
    session-list.png               ← main TUI hero shot
    pr-overview.png                ← P key full-screen view
    diff-view.png                  ← D key diff overlay
    todo-board.png                 ← t key kanban board
    worktree-finish.png            ← W key finish dialog
    pr-badge.png                   ← sidebar PR badge close-up
docs/
  features.md                      ← full feature reference (from README)
  configuration.md                 ← full config.toml reference
  installation.md                  ← install options + requirements
  development.md                   ← build/test/release
  agent-deck-comparison.md         ← comparison table (moved from README)
  plans/                           ← existing internal design docs (untouched)
```

## New README Structure

1. **Header** — title + one-liner tagline
2. **Hero screenshot** — `assets/screenshots/session-list.png`
3. **Features** — 6 highlights, each: 1-sentence description + screenshot + 1-2 line detail
   - Session list with live status indicators → `session-list.png`
   - PR badges + PR overview (`P`) → `pr-badge.png` / `pr-overview.png`
   - Inline diff view (`D`) → `diff-view.png`
   - Todo kanban board (`t`) → `todo-board.png`
   - Worktree-first workflow → `worktree-finish.png`
   - Send text without attaching (`x`) → no screenshot needed (simple modal)
4. **Quick Start** — install command + 4 CLI commands + `hangar hooks install`
5. **Key Bindings** — full table (compact, power-user reference)
6. **Docs** — links to `docs/features.md`, `docs/configuration.md`, `docs/installation.md`

## Screenshots Required

All screenshots at ~160 columns × ~40 rows terminal size.
Store in `assets/screenshots/`. Capture with `screencapture -x` (no shadow).

| File | Key | What to show |
|------|-----|-------------|
| `session-list.png` | — | Hero: 2+ projects, mix of ●◐○ statuses, 2+ PR badges, detail panel, tmux status bar |
| `pr-overview.png` | `P` | 3+ sessions, open + merged states, CI check counts, key hint bar |
| `diff-view.png` | `D` | Diff overlay with 2+ hunks, green `+` / red `-` lines, file header, key hint bar |
| `todo-board.png` | `t` | All 4 columns, cards in 3+ columns, description panel below, one card selected |
| `worktree-finish.png` | `W` | Finish dialog, PR state visible, options/checkboxes |
| `pr-badge.png` | — | Sidebar with green + purple PR badges, oasis_lagoon_dark status pills |

## Docs Content Plan

### `docs/features.md`
Full feature descriptions moved verbatim from README:
- Project-based sessions
- Worktrees first-class
- PR badge in sidebar
- PR overview
- Inline diff view
- Todo kanban board
- PR status in preview
- Send text without attaching
- Lazygit integration
- Status detection
- oasis_lagoon_dark status bar
- Colored session preview

### `docs/configuration.md`
- Full `config.toml` reference with all keys, types, defaults, descriptions
- Projects file format
- Hook installation

### `docs/installation.md`
- Requirements (Go 1.24+, tmux, git, optional: gh, lazygit)
- Install from source (install.sh)
- All install.sh flags
- Post-install setup

### `docs/development.md`
- Build, test, release commands
- GoReleaser usage
- Homebrew formula location

### `docs/agent-deck-comparison.md`
- The comparison table moved from README
- Brief context on the fork relationship
