# Hangar

**A Claude-Code-only terminal session manager.**

Hangar is an opinionated fork of [agent-deck](https://github.com/asheshgoplani/agent-deck) by [@asheshgoplani](https://github.com/asheshgoplani), stripped down to a clean, fast workflow for Claude Code users who live in git repos and worktrees.

---

## What's Different from Agent Deck

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

---

## Features

### Project-based Sessions

Every session belongs to a **Project** — a named git repository with a base directory and base branch.

```bash
hangar project add myrepo ~/code/myrepo        # register a project
hangar project add myrepo ~/code/myrepo main   # explicit base branch
hangar project list
hangar project remove myrepo
```

In the TUI, press `p` to create a new project, or pick from existing ones when opening a new session.

### Worktrees

New sessions automatically create a git worktree under `.worktrees/<branch>` inside the project directory. Before creating the worktree, hangar pulls the base branch so your new session always starts fresh.

```toml
# ~/.hangar/config.toml
[worktree]
auto_update_base = true   # default — pull base branch before worktree creation
```

### Status Detection

Smart polling shows what every Claude session is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| Running | `●` green | Actively working |
| Waiting | `◐` yellow | Needs your input |
| Idle | `○` gray | Ready |
| Error | `✕` red | Something went wrong |

### oasis_lagoon_dark Status Bar

Hangar configures tmux with the oasis_lagoon_dark theme automatically:

- **Status left**: session pill showing fleet status (`⚡ ● 2 │ ◐ 1`)
- **Status right**: session name + current folder + clock, all as rounded powerline pills
- **Window tabs**: rounded pills — orange for active, blue-surface for inactive

The status pill counts **all** sessions including the currently attached one, so you always see the fleet state at a glance.

### Colored Session Preview

Session previews preserve terminal colors (ANSI escape sequences) from the Claude Code TUI, including box-drawing borders and syntax highlighting.

### Lazygit Integration

Press `G` on any session to open **lazygit** in a new tmux window inside that session's tmux session, pointed at the session's working directory.

---

## Key Bindings

| Key | Action |
|-----|--------|
| `Enter` | Attach to session |
| `n` | New session |
| `f` / `F` | Fork (quick / dialog) |
| `p` | New project |
| `G` | Open lazygit for selected session |
| `r` | Rename project or session |
| `M` | Move session to project |
| `S` | Settings |
| `/` | Search |
| `gg` | Jump to top |
| `d` | Delete |
| `?` | Full help |

---

## Installation

**From source:**

```bash
git clone https://github.com/sjoeboo/hangar.git
cd hangar
go install ./cmd/hangar
```

**Requirements:** Go 1.24+, tmux, git. Lazygit optional (for `G` key).

---

## Quick Start

```bash
hangar                                      # Launch TUI
hangar project add myrepo ~/code/myrepo    # Register a project
hangar add ~/code/myrepo -c claude         # Add a session directly
```

---

## Configuration

Config lives at `~/.hangar/config.toml`. Projects are stored in `~/.hangar/projects.toml`.

```toml
[notifications]
enabled = true
minimal = true    # Show ⚡ ● N │ ◐ N format (default)

[worktree]
auto_update_base = true
default_location = "subdirectory"   # places worktrees at repo/.worktrees/<branch>
```

---

## Development

```bash
go build ./...
go test ./...
```

---

## Attribution

Hangar is a fork of [agent-deck](https://github.com/asheshgoplani/agent-deck) by Ashesh Goplani, MIT licensed. Git history is preserved for easy upstream cherry-picks.

Upstream changes incorporated:
- PR #229 / #231: minimal notification mode and status-left restoration on clear
- PR #236: ANSI color preservation in session preview pane

---

## License

MIT — see [LICENSE](LICENSE)
