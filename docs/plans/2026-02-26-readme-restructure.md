# README Restructure + Screenshots Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Split the monolithic README into focused docs, add screenshot placeholders, and rewrite README as a visual landing page.

**Architecture:** Move detailed content verbatim into `docs/` files, rewrite `README.md` from scratch as intro + feature highlights + quickstart, reference screenshots from `assets/screenshots/` (user will provide real PNGs after the structure is in place).

**Tech Stack:** Markdown, GitHub-flavored rendering, `screencapture` for screenshots (macOS).

---

### Task 1: Create directory structure

**Files:**
- Create: `assets/screenshots/.gitkeep`
- Modify: nothing yet

**Step 1: Create the assets/screenshots directory**

```bash
mkdir -p assets/screenshots
touch assets/screenshots/.gitkeep
```

**Step 2: Verify**

```bash
ls assets/screenshots/
```
Expected: `.gitkeep`

**Step 3: Commit**

```bash
git add assets/screenshots/.gitkeep
git commit -m "chore: add assets/screenshots directory for TUI screenshots"
```

---

### Task 2: Create `docs/agent-deck-comparison.md`

**Files:**
- Create: `docs/agent-deck-comparison.md`

**Step 1: Write the file**

Content to write to `docs/agent-deck-comparison.md`:

```markdown
# Hangar vs Agent Deck

Hangar is an opinionated fork of [agent-deck](https://ghe.spotify.net/mnicholson/agent-deck), stripped down to a clean, fast workflow for Claude Code users who live in git repos and worktrees.

| Feature | Agent Deck | Hangar |
|---------|-----------|--------|
| Supported agents | Claude, Gemini, OpenCode, Codex | Claude only |
| MCP management | Socket pool + UI manager | Reads `.mcp.json`, no pooling |
| Skills management | Full UI | Removed |
| Conductor / orchestration | Yes | Removed |
| Web UI | Yes | Removed |
| Groups | Flexible nested groups | Flat projects (name + base dir) |
| New session flow | Free-form path entry | Project picker → worktree |
| Base branch sync | Manual | Auto-pulls base branch on new session |
| Status bar theme | Default tmux | oasis_lagoon_dark (pill tabs, icons) |
| Status pill | Other sessions only | All sessions including current |
| Config dir | `~/.agent-deck/` | `~/.hangar/` |
| Binary | `agent-deck` | `hangar` |

Git history is preserved for easy upstream cherry-picks.
```

**Step 2: Verify it renders correctly**

```bash
cat docs/agent-deck-comparison.md
```

**Step 3: Commit**

```bash
git add docs/agent-deck-comparison.md
git commit -m "docs: add agent-deck comparison doc (moved from README)"
```

---

### Task 3: Create `docs/installation.md`

**Files:**
- Create: `docs/installation.md`

**Step 1: Write the file**

Content to write to `docs/installation.md`:

```markdown
# Installation

## Requirements

| Tool | Version | Required? |
|------|---------|-----------|
| Go | 1.24+ | Required |
| tmux | any | Required |
| git | any | Required |
| `gh` CLI | any | Optional — PR status, PR overview, diff view |
| lazygit | any | Optional — `G` key integration |

## Install from Source

```bash
git clone git@ghe.spotify.net:mnicholson/hangar
cd hangar
./install.sh
```

The installer will:
1. Check for Go (1.24+) and tmux
2. Build from source with version embedding
3. Install to `~/.local/bin/hangar`
4. Configure tmux (mouse, clipboard, 256-color, 50k history)
5. Optionally install Claude Code lifecycle hooks

## Install Options

```bash
./install.sh --dir /usr/local/bin       # custom install dir
./install.sh --skip-tmux-config         # skip tmux setup
./install.sh --skip-hooks               # skip Claude hooks prompt
./install.sh --non-interactive          # CI / unattended install
```

## Post-Install Setup

### Register a project

```bash
hangar project add myrepo ~/code/myrepo        # uses default branch
hangar project add myrepo ~/code/myrepo main   # explicit base branch
```

### Install Claude Code hooks

Hooks enable instant status detection (vs. periodic polling):

```bash
hangar hooks install
hangar hooks status   # verify
```

This writes a hook command to `~/.claude/settings.json`. The hook sends lifecycle events (SessionStart, Stop, UserPromptSubmit, etc.) to Hangar via `~/.hangar/hooks/{id}.json`.

### Launch

```bash
hangar
```
```

**Step 2: Commit**

```bash
git add docs/installation.md
git commit -m "docs: add installation reference doc"
```

---

### Task 4: Create `docs/configuration.md`

**Files:**
- Create: `docs/configuration.md`

**Step 1: Write the file**

Content to write to `docs/configuration.md`:

```markdown
# Configuration

## Config File

Config lives at `~/.hangar/config.toml`. Projects are stored in `~/.hangar/projects.toml`.

```toml
[notifications]
enabled = true
minimal = true    # Show ⚡ ● N │ ◐ N format (default)

[worktree]
auto_update_base = true
default_location = "subdirectory"   # repo/.worktrees/<branch>

[claude]
hooks_enabled = true                # instant status via lifecycle hooks
allow_dangerous_mode = false        # --allow-dangerously-skip-permissions

[tmux]
mouse_mode = true
inject_status_line = true           # set false to keep your own status bar
```

## Key Settings

### `[worktree]`

| Key | Default | Description |
|-----|---------|-------------|
| `auto_update_base` | `true` | Pull base branch before creating a new worktree |
| `default_location` | `"subdirectory"` | Places worktrees at `repo/.worktrees/<branch>` |

### `[claude]`

| Key | Default | Description |
|-----|---------|-------------|
| `hooks_enabled` | `true` | Use lifecycle hooks for instant status detection |
| `allow_dangerous_mode` | `false` | Pass `--allow-dangerously-skip-permissions` to Claude |

### `[tmux]`

| Key | Default | Description |
|-----|---------|-------------|
| `mouse_mode` | `true` | Enable tmux mouse support |
| `inject_status_line` | `true` | Apply oasis_lagoon_dark status bar theme |

Set `inject_status_line = false` to keep your own tmux status bar configuration.

### `[notifications]`

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Show session status in tmux status bar |
| `minimal` | `true` | Show compact `⚡ ● N │ ◐ N` format |

## Projects File

Projects are stored in `~/.hangar/projects.toml`. Managed via CLI:

```bash
hangar project add myrepo ~/code/myrepo        # add project
hangar project add myrepo ~/code/myrepo main   # with explicit base branch
hangar project list                            # list all
hangar project remove myrepo                  # remove
```

## State Database

Todos are stored in `~/.hangar/state.db` (SQLite). This file is managed automatically — no manual editing required.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `HANGAR_INSTANCE_ID` | Current session ID (set by Hangar) |
| `HANGAR_TITLE` | Current session title (set by Hangar) |
| `HANGAR_TOOL` | Tool in use — `claude`, `shell`, etc. (set by Hangar) |
| `HANGAR_PROFILE` | Active profile (set by Hangar) |
```

**Step 2: Commit**

```bash
git add docs/configuration.md
git commit -m "docs: add configuration reference doc"
```

---

### Task 5: Create `docs/development.md`

**Files:**
- Create: `docs/development.md`

**Step 1: Write the file**

Content to write to `docs/development.md`:

```markdown
# Development

## Build & Test

```bash
go build ./...                  # Build all packages
go test ./...                   # Run all tests
go test ./internal/ui/...       # UI tests only
go test ./internal/session/...  # Session tests
go run ./cmd/hangar              # Run locally without installing
make install-user               # Build + install to ~/.local/bin
```

> **Note:** Two tests in `internal/ui/` have pre-existing failures:
> - `TestNewDialog_WorktreeToggle_ViaKeyPress`
> - `TestNewDialog_TypingResetsSuggestionNavigation`

## Architecture

```
cmd/hangar/          # CLI entry point and subcommands
internal/
  session/           # Core session model, config, hooks, status detection, todos
  statedb/           # SQLite state (todos table)
  tmux/              # tmux interaction layer (status, capture, send-keys)
  ui/                # Bubble Tea TUI (home.go is the main model)
  git/               # Git worktree operations + diff fetching
  update/            # Self-update logic
  profile/           # Multi-profile support
  logging/           # Structured logging
```

Key files: `internal/ui/home.go` (main TUI model), `internal/ui/styles.go` (theme), `internal/session/instance.go` (session model).

## Release

Releases are built via [GoReleaser](https://goreleaser.com):

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean   # Test build (no publish)
goreleaser release                       # Full release (requires GITHUB_TOKEN)
```

Release artifacts: `hangar_{version}_{os}_{arch}.tar.gz`

Homebrew formula: [mnicholson/homebrew-tap](https://github.com/mnicholson/homebrew-tap)
```

**Step 2: Commit**

```bash
git add docs/development.md
git commit -m "docs: add development reference doc"
```

---

### Task 6: Create `docs/features.md`

**Files:**
- Create: `docs/features.md`

This is the largest file — full feature reference moved from README.

**Step 1: Write the file**

Content to write to `docs/features.md`:

```markdown
# Features

## Project-Based Sessions

Every session belongs to a **Project** — a named git repository with a base directory and base branch. Press `p` to create a new project from the TUI, or manage from the CLI:

```bash
hangar project add myrepo ~/code/myrepo        # register a project
hangar project add myrepo ~/code/myrepo main   # explicit base branch
hangar project list
hangar project remove myrepo
```

When opening a new session (`n`), you pick a project first. Hangar automatically creates a git worktree and pulls the latest base branch before starting.

## Worktrees First-Class

New sessions automatically create a git worktree under `.worktrees/<branch>` inside the project directory. The base branch is pulled fresh each time so you always start from a clean state. Worktree creation runs **asynchronously** — the UI stays responsive even for large monorepos.

```toml
# ~/.hangar/config.toml
[worktree]
auto_update_base = true   # default — pull base branch before worktree creation
default_location = "subdirectory"   # places worktrees at repo/.worktrees/<branch>
```

Press `W` on a worktree session to open the finish dialog: merge branch, remove worktree, and delete session in one step. The dialog also shows the current PR state so you can confirm before merging.

## PR Badge in Sidebar

Worktree sessions with an open, merged, or closed PR display a color-coded badge directly in the session list:

```
  ● my-feature-session [#456]
  ◐ fix-auth-bug [#789]
  ○ another-branch
```

Badges are green for open, purple for merged, and red for closed. Draft PRs don't show a badge.

## PR Overview (`P`)

Press **`P`** to enter a full-screen PR overview showing every session that has an associated pull request:

```
  PR       STATE     CHECKS          SESSION
─────────────────────────────────────────────────────
  #123     open      ✓12 ●2         my-feature
  #456     merged    ✓8             fix-auth
  #789     open      ✗1 ✓4         refactor-api
─────────────────────────────────────────────────────
  Enter Attach · o Open PR · r Refresh · P/Esc Back · ↑↓/jk Nav
```

Inside PR overview: `Enter` attaches to the session, `o` opens the PR in the browser, `r` force-refreshes the PR data, `j/k` or `↑/↓` navigate.

## Inline Diff View (`D`)

Press **`D`** on any worktree session to open a pager-style diff overlay showing unstaged and staged changes for that session's working directory:

```
  Diff: 3 files, +47 -12
────────────────────────────────────────────────────
  internal/ui/home.go
  @@ -45,8 +45,10 @@ func main() @@
  -	old line
  +	new line
     context line
────────────────────────────────────────────────────
  j/k scroll · space/b page · ctrl+d/u half-page · g/G top/bottom · e editor · q close
```

Full pager key bindings: `j/k` (1 line), `Space/b` (full page), `Ctrl+D/U` (half page), `g/G` (top/bottom), `e` opens the file at the current hunk in `$EDITOR`, `q`/`Esc`/`D` closes.

## Todo Kanban Board (`t`)

Press **`t`** to open a per-project kanban board. Todos are organized into columns by status:

```
  project-name

  todo (3)       in progress (1)    in review (0)    done (2)
  ──────────     ──────────────     ────────────     ──────────
  ○ Build API    ● Add logging      (empty)          ✓ Add tests
  ○ Write docs                                       ✓ Deploy

  description
  ────────────────────────────────────────────────────
  Build the REST API endpoints for user authentication.

  ↑↓ card · ←→ col · n new · e edit · d delete · s status · shift+←/→ move · enter open · esc close
```

- `←/→` or `h/l` — navigate columns; `↑/↓` or `j/k` — navigate cards
- `n` — new todo (pre-selects the current column's status)
- `e` — edit selected todo
- `d` — delete selected todo
- `s` — change status (status picker)
- `Shift+←/→` — move card left/right to adjacent column
- `Enter` — create a new Claude Code session + worktree from the selected todo

The description panel below the board shows the full text of the selected todo.

Todos are stored in `~/.hangar/state.db` (SQLite). Lifecycle hooks automatically update todo status when you open a PR, get reviews, or finish a worktree.

## PR Status in Preview

When a worktree session has an open GitHub PR, the preview pane shows:

```
PR  #42 · main · feat: add thing
     CI  ✓ 12 passed
```

PR info is fetched via `gh pr view` with a 60-second TTL cache. Press **`o`** to open the PR URL in your browser. Press **`Ctrl+R`** to force-refresh git and PR status for the selected session.

## Send Text Without Attaching (`x`)

Press **`x`** on any session to open a send-text modal. Type a message and press Enter — it's delivered to the session without you having to attach and detach.

## Lazygit Integration (`G`)

Press **`G`** on any session to open **lazygit** in a new tmux window inside that session's tmux session, pointed at the session's working directory.

## Status Detection

Smart polling shows what every Claude session is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| Running | `●` green | Actively working |
| Waiting | `◐` yellow | Needs your input |
| Idle | `○` gray | Ready |
| Error | `✕` red | Something went wrong |

Status is detected via Claude Code lifecycle hooks (fsnotify-based, instant) with tmux content polling as fallback. Install hooks with:

```bash
hangar hooks install
```

## oasis_lagoon_dark Status Bar

Hangar configures tmux with the oasis_lagoon_dark theme automatically:

- **Status left**: fleet pill showing `⚡ ● 2 │ ◐ 1` (all sessions, including current)
- **Status right**: session name + folder + clock as rounded powerline pills
- **Window tabs**: rounded pills — orange active, blue-surface inactive

## Colored Session Preview

Session previews preserve ANSI escape sequences from the Claude Code TUI, including box-drawing borders and syntax highlighting.
```

**Step 2: Commit**

```bash
git add docs/features.md
git commit -m "docs: add full feature reference doc (content from README)"
```

---

### Task 7: Rewrite `README.md`

**Files:**
- Modify: `README.md` (full rewrite)

This is the most important task. Write the new README from scratch. Screenshots are referenced as `assets/screenshots/*.png` — the files don't exist yet (user will provide them), but the references should be in place.

**Step 1: Write the new README**

Replace the entire content of `README.md` with:

```markdown
# Hangar

**A terminal session manager for Claude Code — built for engineers who live in git worktrees.**

![Session list](assets/screenshots/session-list.png)

---

## Features

### Live Session Monitoring

Every Claude session shows real-time status — running `●`, waiting `◐`, or idle `○` — updated instantly via Claude Code lifecycle hooks.

![Session list with status indicators](assets/screenshots/session-list.png)

### PR Badges & Overview

Worktree sessions with open PRs display color-coded badges in the sidebar (green = open, purple = merged). Press **`P`** for a full-screen PR dashboard with CI check status.

![PR overview](assets/screenshots/pr-overview.png)

### Inline Diff View

Press **`D`** to open a pager-style diff overlay for any session's working directory — scroll, page, and jump to hunks without leaving Hangar.

![Inline diff view](assets/screenshots/diff-view.png)

### Todo Kanban Board

Press **`t`** to open a per-project kanban board with todo → in progress → in review → done columns. Press `Enter` on a card to spin up a new Claude session + worktree for that task.

![Todo kanban board](assets/screenshots/todo-board.png)

### Worktree-First Workflow

Every new session automatically creates a git worktree from the latest base branch. Press **`W`** to merge, clean up, and close in one step.

![Worktree finish dialog](assets/screenshots/worktree-finish.png)

### Send Text Without Attaching

Press **`x`** to send a message to any session — no attach/detach needed.

---

## Quick Start

```bash
git clone git@ghe.spotify.net:mnicholson/hangar
cd hangar
./install.sh

# Register a project
hangar project add myrepo ~/code/myrepo

# Install Claude Code hooks (enables instant status detection)
hangar hooks install

# Launch
hangar
```

**Requirements:** Go 1.24+, tmux, git. `gh` CLI optional (PR status, diff view). lazygit optional (`G` key).

---

## Key Bindings

| Key | Action |
|-----|--------|
| `Enter` | Attach to session |
| `n` / `N` | New session / Quick session |
| `p` | New project |
| `x` | Send text to session |
| `o` | Open PR in browser |
| `G` | Open lazygit |
| `W` | Worktree finish dialog |
| `D` | Inline diff view |
| `P` | PR overview |
| `t` | Todo kanban board |
| `r` | Rename project or session |
| `M` | Move session to project |
| `S` | Settings |
| `/` | Search |
| `~` | Toggle status sort |
| `Ctrl+R` | Force-refresh git/PR status |
| `d` | Delete |
| `Ctrl+Z` | Undo delete |
| `?` | Full help |
| `Ctrl+Q` | Detach from session |
| `q` | Quit |

---

## Documentation

- [Full Feature Reference](docs/features.md)
- [Configuration](docs/configuration.md)
- [Installation](docs/installation.md)
- [Development](docs/development.md)

---

## License

MIT — see [LICENSE](LICENSE)
```

**Step 2: Verify the README renders as expected**

Check that all section headers are correct and all image references use the right paths:

```bash
grep "assets/screenshots" README.md
```

Expected: 5 lines referencing screenshot files.

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README as visual landing page with screenshot placeholders"
```

---

### Task 8: Add screenshot capture instructions

**Files:**
- Create: `assets/screenshots/README.md`

This file tells anyone (including you in the future) exactly what each screenshot should show. It doubles as the checklist for capturing them.

**Step 1: Write the file**

Content to write to `assets/screenshots/README.md`:

```markdown
# Screenshots

These screenshots are referenced from the main README. Capture at **~160 columns × ~40 rows** terminal size using `screencapture -x` (macOS, no drop shadow).

## Checklist

### `session-list.png` — Hero shot
- [ ] 2+ projects visible in sidebar
- [ ] Mix of statuses: at least one `●` green, one `◐` yellow, one `○` gray
- [ ] 2+ sessions with PR badges (`[#123]` style) — green and purple
- [ ] Detail/preview panel visible on the right showing session content
- [ ] tmux status bar visible (oasis_lagoon_dark theme with fleet pill)

```bash
screencapture -x assets/screenshots/session-list.png
```

### `pr-overview.png` — PR Overview (`P` key)
- [ ] Full-screen PR overview (press `P` from session list)
- [ ] 3+ sessions listed with PR numbers
- [ ] Mix of states: open (green) and merged (purple)
- [ ] CI check counts visible (`✓N ●N` etc.)
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/pr-overview.png
```

### `diff-view.png` — Inline Diff (`D` key)
- [ ] Diff overlay open over the session list (press `D` on a worktree session)
- [ ] 2+ hunks visible with real file paths
- [ ] Green `+` additions and red `-` deletions clearly visible
- [ ] File name header and change count (`N files, +N -N`)
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/diff-view.png
```

### `todo-board.png` — Todo Kanban (`t` key)
- [ ] All 4 columns: todo, in progress, in review, done
- [ ] Cards in at least 3 columns (doesn't need to be all 4)
- [ ] One card visibly selected/highlighted
- [ ] Description panel below showing the selected card's text
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/todo-board.png
```

### `worktree-finish.png` — Worktree Finish Dialog (`W` key)
- [ ] Finish dialog open (press `W` on a worktree session)
- [ ] PR state visible in the dialog
- [ ] Options/checkboxes for merge + cleanup visible

```bash
screencapture -x assets/screenshots/worktree-finish.png
```

## Tips

- Use a clean terminal with a readable font (JetBrains Mono, Fira Code, etc.)
- Use real project/branch names — placeholder text looks less authentic
- If using iTerm2, disable transparency before capturing
- Crop to the terminal content only (no window chrome needed)
```

**Step 2: Commit**

```bash
git add assets/screenshots/README.md
git commit -m "docs: add screenshot capture checklist and instructions"
```

---

### Task 9: Final verification

**Step 1: Confirm all links in README resolve**

```bash
# Check docs links
ls docs/features.md docs/configuration.md docs/installation.md docs/development.md

# Check screenshot paths match what README references
grep "assets/screenshots" README.md
ls assets/screenshots/
```

**Step 2: Check no broken internal links remain**

```bash
grep -r "What's Different from Agent Deck" README.md
```

Expected: no output (that section was moved to `docs/agent-deck-comparison.md`).

**Step 3: Final commit if anything was missed**

```bash
git status
# If clean: nothing to do
# If dirty: git add <files> && git commit -m "docs: fix any remaining issues"
```

---

## Screenshot Handoff

After Task 9 completes, the repo structure is ready. Provide the following to the user:

**To capture screenshots, run hangar and use these commands:**

```bash
# For each view, get into position then:
screencapture -x /path/to/hangar/assets/screenshots/<name>.png
```

See `assets/screenshots/README.md` for the full checklist of what each screenshot should show.

Once the PNG files are added to `assets/screenshots/`, they'll appear automatically in the README — no further changes needed.
