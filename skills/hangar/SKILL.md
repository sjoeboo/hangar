---
name: hangar
description: Terminal session manager for AI coding agents. Use when user mentions "hangar", "session", "sub-agent", "MCP attach", "git worktree", or needs to (1) create/start/stop/restart/fork sessions, (2) attach/detach MCPs, (3) manage projects/profiles, (4) get session output, (5) configure hangar, (6) troubleshoot issues, (7) launch sub-agents, or (8) create/manage worktree sessions. Covers CLI commands, TUI shortcuts, config.toml options, and automation.
compatibility: claude, opencode
---

# Hangar

Terminal session manager for AI coding agents. Built with Go + Bubble Tea.

**Version:** 0.8.98 | **Repo:** [github.com/sjoeboo/hangar](https://github.com/sjoeboo/hangar)

## Script Path Resolution (IMPORTANT)

This skill includes helper scripts in its `scripts/` subdirectory. When Claude Code loads this skill, it shows a line like:

```
Base directory for this skill: /path/to/.../skills/hangar
```

**You MUST use that base directory path to resolve all script references.** Store it as `SKILL_DIR`:

```bash
# Set SKILL_DIR to the base directory shown when this skill was loaded
SKILL_DIR="/path/shown/in/base-directory-line"

# Then run scripts as:
$SKILL_DIR/scripts/launch-subagent.sh "Title" "Prompt" --wait
```

**Common mistake:** Do NOT use `<project-root>/scripts/launch-subagent.sh`. The scripts live inside the skill's own directory (plugin cache or project skills folder), NOT in the user's project root.

**For plugin users**, the path looks like: `~/.claude/plugins/cache/hangar/hangar/<hash>/skills/hangar/scripts/`
**For local development**, the path looks like: `<repo>/skills/hangar/scripts/`

## Quick Start

```bash
# Launch TUI
hangar

# Create and start a session
hangar add -t "Project" -c claude /path/to/project
hangar session start "Project"

# Send message and get output
hangar session send "Project" "Analyze this codebase"
hangar session output "Project"
```

## Essential Commands

| Command | Purpose |
|---------|---------|
| `hangar` | Launch interactive TUI |
| `hangar add -t "Name" -c claude /path` | Create session |
| `hangar session start/stop/restart <name>` | Control session |
| `hangar session send <name> "message"` | Send message |
| `hangar session output <name>` | Get last response |
| `hangar session current [-q\|--json]` | Auto-detect current session |
| `hangar session fork <name>` | Fork Claude conversation |
| `hangar mcp list` | List available MCPs |
| `hangar mcp attach <name> <mcp>` | Attach MCP (then restart) |
| `hangar status` | Quick status summary |
| `hangar add --worktree <branch>` | Create session in git worktree |
| `hangar worktree list` | List worktrees with sessions |
| `hangar worktree cleanup` | Find orphaned worktrees/sessions |

**Status:** `●` running | `◐` waiting | `○` idle | `✕` error

## Sub-Agent Launch

**Use when:** User says "launch sub-agent", "create sub-agent", "spawn agent"

```bash
$SKILL_DIR/scripts/launch-subagent.sh "Title" "Prompt" [--mcp name] [--wait]
```

The script auto-detects current session/profile and creates a child session.

### Retrieval Modes

| Mode | Command | Use When |
|------|---------|----------|
| **Fire & forget** | (no --wait) | Default. Tell user: "Ask me to check when ready" |
| **On-demand** | `hangar session output "Title"` | User asks to check |
| **Blocking** | `--wait` flag | Need immediate result |

### Recommended MCPs

| Task Type | MCPs |
|-----------|------|
| Web research | `exa`, `firecrawl` |
| Code documentation | `context7` |
| Complex reasoning | `sequential-thinking` |

## Sub-Agent Options

```bash
$SKILL_DIR/scripts/launch-subagent.sh "Title" "Prompt" \
  --path /project/dir \     # Working directory (auto-inherits parent path if omitted)
  --wait \                  # Block until response is ready
  --timeout 180 \           # Seconds to wait (default: 300)
  --mcp exa                 # Attach MCP servers (can repeat)
```

## TUI Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move up/down |
| `Enter` | Attach to session |

### Session Actions
| Key | Action |
|-----|--------|
| `n` | New session |
| `r` | Rename project or session |
| `f/F` | Fork Claude session |
| `d` | Delete |
| `M` | Move to project |
| `G` | Open lazygit for selected session |

### Search & Filter
| Key | Action |
|-----|--------|
| `/` | Local search |
| `!@#$` | Filter by status (running/waiting/idle/error) |

### Global
| Key | Action |
|-----|--------|
| `?` | Help overlay |
| `Ctrl+Q` | Detach (keep tmux running) |
| `q` | Quit |

## MCP Management

**Default:** Do NOT attach MCPs unless user explicitly requests.

```bash
# List available
hangar mcp list

# Attach and restart
hangar mcp attach <session> <mcp-name>
hangar session restart <session>

# Or attach on create
hangar add -t "Task" -c claude --mcp exa /path
```

**Scopes:**
- **LOCAL** (default) - `.mcp.json` in project, affects only that session
- **GLOBAL** (`--global`) - Claude config, affects all projects

## Worktree Workflows

### Create Session in Git Worktree

When working on a feature that needs isolation from main branch:

```bash
# Create session with new worktree and branch
hangar add /path/to/repo -t "Feature Work" -c claude --worktree feature/my-feature --new-branch

# Create session in existing branch's worktree
hangar add . --worktree develop -c claude
```

### List and Manage Worktrees

```bash
# List all worktrees and their associated sessions
hangar worktree list

# Show detailed info for a session's worktree
hangar worktree info "My Session"

# Find orphaned worktrees/sessions (dry-run)
hangar worktree cleanup

# Actually clean up orphans
hangar worktree cleanup --force
```

### When to Use Worktrees

| Use Case | Benefit |
|----------|---------|
| **Parallel agent work** | Multiple agents on same repo, different branches |
| **Feature isolation** | Keep main branch clean while agent experiments |
| **Code review** | Agent reviews PR in worktree while main work continues |
| **Hotfix work** | Quick branch off main without disrupting feature work |

## Configuration

**File:** `~/.hangar/config.toml`

```toml
[claude]
config_dir = "~/.claude-work"    # Custom Claude profile
dangerous_mode = true            # --dangerously-skip-permissions

[logs]
max_size_mb = 10                 # Max before truncation
max_lines = 10000                # Lines to keep

[mcps.exa]
command = "npx"
args = ["-y", "exa-mcp-server"]
env = { EXA_API_KEY = "key" }
description = "Web search"
```

See [config-reference.md](references/config-reference.md) for all options.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Session shows error | `hangar session start <name>` |
| MCPs not loading | `hangar session restart <name>` |
| Flag not working | Put flags BEFORE arguments: `-m "msg" name` not `name -m "msg"` |

### Get Help

- **GitHub Issues:** For bug reports and feature requests

### Report a Bug

If something isn't working, create a GitHub issue with context:

```bash
# Gather debug info
hangar version
hangar status --json
cat ~/.hangar/config.toml | grep -v "KEY\|TOKEN\|SECRET"  # Sanitized config

# Create issue at:
# https://github.com/sjoeboo/hangar/issues/new
```

**Include:**
1. What you tried (command/action)
2. What happened vs expected
3. Output of commands above
4. Relevant log: `tail -100 ~/.hangar/logs/hangar_<session>_*.log`

See [troubleshooting.md](references/troubleshooting.md) for detailed diagnostics.

## Session Sharing

Share Claude sessions between developers for collaboration or handoff.

**Use when:** User says "share session", "export session", "send to colleague", "import session"

```bash
# Export current session to file (session-share is a sibling skill)
$SKILL_DIR/../session-share/scripts/export.sh
# Output: ~/session-shares/session-<date>-<title>.json

# Import received session
$SKILL_DIR/../session-share/scripts/import.sh ~/Downloads/session-file.json
```

**See:** [session-share skill](../session-share/SKILL.md) for full documentation.

## Critical Rules

1. **Flags before arguments:** `session start -m "Hello" name` (not `name -m "Hello"`)
2. **Restart after MCP attach:** Always run `session restart` after `mcp attach`
3. **Never poll from other agents** - can interfere with target session

## References

- [cli-reference.md](references/cli-reference.md) - Complete CLI command reference
- [config-reference.md](references/config-reference.md) - All config.toml options
- [tui-reference.md](references/tui-reference.md) - TUI features and shortcuts
- [troubleshooting.md](references/troubleshooting.md) - Common issues and bug reporting
- [session-share skill](../session-share/SKILL.md) - Export/import sessions for collaboration
