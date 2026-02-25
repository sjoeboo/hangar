# Hangar

**A Claude-Code-only terminal session manager.**

Hangar is an opinionated fork of [agent-deck](https://github.com/sjoeboo/agent-deck) by @sjoeboo, stripped down to a clean, fast workflow for Claude Code users who live in git repos and worktrees.

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

Every session belongs to a **Project** — a named git repository with a base directory and base branch. Press `p` to create a new project from the TUI, or manage from the CLI:

```bash
hangar project add myrepo ~/code/myrepo        # register a project
hangar project add myrepo ~/code/myrepo main   # explicit base branch
hangar project list
hangar project remove myrepo
```

When opening a new session (`n`), you pick a project first. Hangar automatically creates a git worktree and pulls the latest base branch before starting.

### Worktrees First-class

New sessions automatically create a git worktree under `.worktrees/<branch>` inside the project directory. The base branch is pulled fresh each time so you always start from a clean state.

```toml
# ~/.hangar/config.toml
[worktree]
auto_update_base = true   # default — pull base branch before worktree creation
default_location = "subdirectory"   # places worktrees at repo/.worktrees/<branch>
```

Press `W` on a worktree session to open the finish dialog: merge branch, remove worktree, and delete session in one step.

### PR Status in Preview

When a worktree session has an open GitHub PR, the preview pane shows:

```
PR  #42 · main · feat: add thing
     CI  ✓ 12 passed
```

PR info is fetched via `gh pr view` with a 60-second TTL cache. Press **`o`** to open the PR URL in your browser.

### Send Text Without Attaching

Press **`x`** on any session to open a send-text modal. Type a message and press Enter — it's delivered to the session without you having to attach and detach.

### Lazygit Integration

Press **`G`** on any session to open **lazygit** in a new tmux window inside that session's tmux session, pointed at the session's working directory.

### Status Detection

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

### oasis_lagoon_dark Status Bar

Hangar configures tmux with the oasis_lagoon_dark theme automatically:

- **Status left**: fleet pill showing `⚡ ● 2 │ ◐ 1` (all sessions, including current)
- **Status right**: session name + folder + clock as rounded powerline pills
- **Window tabs**: rounded pills — orange active, blue-surface inactive

### Colored Session Preview

Session previews preserve ANSI escape sequences from the Claude Code TUI, including box-drawing borders and syntax highlighting.

---

## Key Bindings

| Key | Action |
|-----|--------|
| `Enter` | Attach to session |
| `n` / `N` | New session / Quick session |
| `f` / `F` | Fork (quick / dialog) |
| `p` | New project |
| `x` | Send text to session |
| `o` | Open PR in browser (when PR detected) |
| `G` | Open lazygit for selected session |
| `W` | Worktree finish dialog |
| `r` | Rename project or session |
| `M` | Move session to project |
| `S` | Settings |
| `/` | Search |
| `gg` | Jump to top |
| `G` | Lazygit |
| `d` | Delete |
| `Ctrl+Z` | Undo delete |
| `?` | Full help |

---

## Installation

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

**Options:**

```bash
./install.sh --dir /usr/local/bin       # custom install dir
./install.sh --skip-tmux-config         # skip tmux setup
./install.sh --skip-hooks               # skip Claude hooks prompt
./install.sh --non-interactive          # CI / unattended install
```

**Requirements:** Go 1.24+, tmux, git. `gh` CLI optional (for PR status). Lazygit optional (for `G` key).

---

## Quick Start

```bash
hangar                                        # Launch TUI
hangar project add myrepo ~/code/myrepo      # Register a project
hangar add ~/code/myrepo -c claude           # Add a session directly
hangar session send "my session" "hello"     # Send text from CLI
hangar hooks status                          # Check hook installation
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
default_location = "subdirectory"   # repo/.worktrees/<branch>

[claude]
hooks_enabled = true                # instant status via lifecycle hooks
allow_dangerous_mode = false        # --allow-dangerously-skip-permissions

[tmux]
mouse_mode = true
inject_status_line = true           # set false to keep your own status bar
```

---

## Development

```bash
go build ./...
go test ./...
make install-user   # build + install to ~/.local/bin
```

---

## Attribution

Hangar is a fork of [agent-deck](https://github.com/sjoeboo/agent-deck) by @sjoeboo, MIT licensed. Git history is preserved for easy upstream cherry-picks.

---

## License

MIT — see [LICENSE](LICENSE)
