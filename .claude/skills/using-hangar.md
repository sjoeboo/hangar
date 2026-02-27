---
name: using-hangar
description: >
  Use when helping users with Hangar — the terminal session manager for AI coding agents.
  Covers session management, worktrees, groups, hooks, and common workflows.
---

# Using Hangar

**Hangar** is a terminal session manager for AI coding agents (Claude Code, Gemini CLI, OpenCode, Codex). It runs as a TUI on top of tmux and provides session monitoring, worktree management, project groups, and lifecycle hooks.

## Quick Reference

```bash
hangar                        # Open TUI
hangar add [path]             # Add session for a project
hangar add [path] -w          # Add session with a new git worktree
hangar launch [path]          # Add + start session immediately
hangar list                   # List sessions (JSON)
hangar session attach <id>    # Attach to a session
hangar session stop <id>      # Stop a session
hangar session send <id> <msg>  # Send a message to a running session
hangar hooks install          # Install Claude Code lifecycle hooks
```

## Session Lifecycle

```
hangar add [path]             # Create session record
  → session start <id>        # Launch in tmux
  → (work happens)
  → session stop <id>         # Stop the tmux process
  → hangar remove <id>        # Remove from Hangar
```

Or all-in-one: `hangar launch [path]` (creates + starts).

## TUI Navigation

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Navigate session list |
| `enter` | Attach to selected session |
| `n` | New session dialog |
| `p` | New project/group |
| `x` | Send message to session |
| `M` | Move session to group |
| `r` | Rename |
| `R` | Restart session |
| `d` | Delete (with undo) |
| `D` | View git diff (worktree sessions) |
| `W` | Worktree finish (archive + cleanup) |
| `P` | PR overview (all sessions with open PRs) |
| `t` | Todo kanban board |
| `o` | Open PR in browser |
| `G` | Open lazygit |
| `S` | Settings |
| `/` | Search sessions |
| `~` | Toggle status sort |
| `Ctrl+R` | Force-refresh git/PR status |
| `?` | Help |
| `Ctrl+Q` | Detach (return to hangar TUI) |
| `q` | Quit hangar |

## Worktree Workflow

Worktrees let you run multiple isolated branches of the same repo simultaneously.

```bash
# Create session with new worktree
hangar add /path/to/repo -w -b feature/my-branch

# Or use the TUI: press 'n', toggle worktree mode
# The session gets its own branch and working directory

# When done: press 'W' in TUI or:
hangar worktree finish <session-id>
  # This archives the branch (does NOT merge to main)
  # PRs are the intended delivery mechanism
```

**Key rule**: Hangar never merges worktrees into main/master directly. Always use a PR.

## Groups & Projects

Sessions are organized into projects (groups). Each project maps to a path on disk.

```bash
hangar project add /path/to/repo    # Register a project
hangar project list                  # List registered projects
```

In the TUI:
- Press `p` to create a new group
- Press `M` to move a session into a group
- Groups appear as tree nodes above their sessions

## Hook Installation

Hooks let Hangar detect Claude Code session status instantly (without polling):

```bash
hangar hooks install    # Installs hook commands into ~/.claude/settings.json
hangar hooks status     # Check if hooks are installed correctly
hangar hooks remove     # Remove hooks
```

After installing hooks, Hangar writes to `~/.hangar/hooks/{id}.json` on each Claude Code lifecycle event (SessionStart, Stop, PermissionRequest, etc.).

## Session Status Indicators

| Status | Meaning |
|--------|---------|
| Waiting / ● | Session is idle, waiting for input |
| Running / spinner | Agent is actively working |
| Needs input | Claude is waiting for a permission grant |
| Error | Session crashed or exited unexpectedly |

Status is detected via hooks (fast) or tmux pane content scanning (fallback).

## Multi-Profile Support

Hangar supports multiple profiles (e.g., work vs personal):

```bash
hangar --profile work add /path/to/repo
hangar --profile personal list
```

Profile data is stored in `~/.hangar/profiles/`.

## Sending Messages to Sessions

```bash
hangar session send <id> "implement the login page"
# OR in the TUI: select session, press 'x'
```

The message is typed into the session as if you typed it at the terminal. Claude Code picks it up as input.

## Todo Board

Press `t` in the TUI to open the Kanban todo board for the selected project.

Columns: **Backlog** → **In Progress** → **Done**

Todos are stored in SQLite at `~/.hangar/hangar.db` and are per-project (based on path).

## Experiments / Quick Sessions

```bash
hangar try my-experiment        # Create dated experiment folder + session
hangar try list                 # List existing experiments
```

Creates `~/src/tries/2026-02-26-my-experiment/` (date prefix configurable in settings).

## Environment Variables (inside a session)

When running inside a Hangar-managed session, these are set automatically:

| Variable | Value |
|----------|-------|
| `HANGAR_INSTANCE_ID` | Unique session ID |
| `HANGAR_TITLE` | Session display name |
| `HANGAR_TOOL` | Tool name (claude, gemini, shell, etc.) |
| `HANGAR_PROFILE` | Active profile name |

## Troubleshooting

### Session shows wrong status
1. Press `Ctrl+R` to force-refresh
2. Check hooks: `hangar hooks status` — if not installed, install them
3. Check logs: `tail -f ~/.hangar/logs/hangar.log`

### Hooks not working
```bash
hangar hooks status     # Should show all hooks as installed
hangar hooks install    # Re-install if any are missing
```

### Session won't attach
```bash
hangar session show <id>    # Check if tmux session exists
tmux ls                      # List all tmux sessions (hangar_ prefix)
```

### Database corruption
```bash
ls -la ~/.hangar/hangar.db   # Check file exists and has size > 0
# SQLite uses WAL mode — if both hangar.db-wal and hangar.db-shm exist, it's mid-write
# Wait a moment and retry; never delete -wal/-shm files while hangar is running
```

### Two hangar instances running
Hangar uses heartbeat-based primary election. Only one instance owns writes at a time. If the TUI feels stale, it may be a secondary instance. Quit and relaunch.

## Configuration

Settings are in `~/.hangar/` and managed via the TUI (`S` key).

Key settings:
- **Notification style**: minimal (icon+count) or detailed (with names)
- **Theme**: automatically follows OS dark/light mode
- **Worktree base directory**: where worktrees are created relative to repo
- **Experiments directory**: base for `hangar try` sessions

## Common Patterns for Agents

If you are an agent running inside Hangar:
- Your session ID is `$HANGAR_INSTANCE_ID`
- You can check your own status file: `cat ~/.hangar/hooks/$HANGAR_INSTANCE_ID.json`
- To send yourself a notification trigger, write a compatible hook file
- The TUI will update within 100ms via fsnotify

## CLI Reference Summary

```bash
hangar add [path] [flags]         # Create session
  -t <title>                      # Custom title
  -c <tool>                       # Tool: claude, gemini, codex, shell
  -b <branch>                     # Branch (worktree mode)
  -w                              # Enable worktree mode
  --mcp <name>                    # Add MCP server
  --cmd <command>                  # Custom command override

hangar launch [path] [flags]      # Create + start session
  -m <message>                    # Initial message to send
  --no-wait                       # Don't wait for session to start

hangar session <subcommand>
  start <id>                      # Start (launch tmux)
  stop <id>                       # Stop (kill tmux)
  restart <id>                    # Restart with same config
  attach <id>                     # Attach interactively
  show <id>                       # Show session details (JSON)
  send <id> <message>             # Send message
  fork <id>                       # Fork session into worktree
  set <id> <field> <value>        # Update field (title, path, command, tool)
  current                         # Get current session (from HANGAR_INSTANCE_ID)

hangar worktree
  list                            # List sessions with worktrees
  info <id>                       # Worktree details
  cleanup                         # Remove orphaned worktrees
  finish <id> [flags]             # Archive branch + cleanup
    --target <branch>             # Target branch for context (not merged into)
    --force                       # Skip confirmation

hangar try <name>                 # Quick experiment session
hangar project add/list/remove    # Project management
hangar hooks install/remove/status  # Lifecycle hook management
hangar update                     # Self-update hangar binary
hangar version                    # Show version info
```
