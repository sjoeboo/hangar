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
