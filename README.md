# Hangar

**A terminal session manager for Claude Code — built for engineers who live in git worktrees.**

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
