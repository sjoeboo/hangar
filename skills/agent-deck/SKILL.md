---
name: agent-deck
description: Complete guide for managing AI coding agent sessions via agent-deck CLI. Use when needing to operate agent-deck from command line for (1) creating, starting, stopping, restarting, or forking Claude/AI sessions, (2) managing MCP (Model Context Protocol) servers - attaching/detaching MCPs locally or globally, (3) organizing sessions with groups and hierarchies, (4) creating sub-sessions with parent-child relationships, (5) managing multiple profiles for different contexts (personal/work/client), (6) checking session status and finding waiting sessions, (7) retrieving session outputs/responses from sub-agents, (8) automating session management with JSON output and scripting, or (9) any other agent-deck CLI operations including session lifecycle management, MCP configuration, group organization, sub-session nesting, and profile separation.
---

# Agent Deck CLI

Terminal session manager for AI coding agents. CLI provides full control over sessions, MCPs, groups, and profiles.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `agent-deck add -t "Name" -c claude /path` | Create session |
| `agent-deck session start "Name"` | Start session (creates tmux) |
| `agent-deck session send "Name" "prompt"` | Send message |
| `agent-deck session output "Name"` | Get last response |
| `agent-deck session show "Name"` | Show details |
| `agent-deck session current` | Auto-detect current session |
| `agent-deck mcp list` | List available MCPs |
| `agent-deck status` | Quick status summary |

**Status indicators:** `●` running | `◐` waiting | `○` idle | `✕` error

## Essential Workflows

### Workflow 1: Quick Session (Default)

**Always complete the full workflow:** `add` -> `start` -> `send`

```bash
mkdir -p /tmp/task && \
  agent-deck add -t "Task Name" -g group -c claude /tmp/task && \
  agent-deck session start "Task Name" && \
  agent-deck session send "Task Name" "Prompt derived from user request"
```

**With MCPs (only when user requests):**
```bash
agent-deck mcp list  # Always list first
agent-deck add -t "Task" -c claude --mcp <mcp-name> /tmp/task && \
  agent-deck session start "Task" && \
  agent-deck session send "Task" "prompt"
```

### Workflow 2: Background Agent Orchestration (RECOMMENDED)

**For research tasks:** Use Claude Code's Task tool with `run_in_background` to handle the entire workflow cleanly.

```
Task(
  subagent_type: "general-purpose",
  run_in_background: true,
  prompt: "Complete this workflow and return the response:

    1. Create and start session:
    mkdir -p /tmp/<name> && \
      agent-deck add -t '<Title>' -g research -c claude /tmp/<name> && \
      agent-deck session start '<Title>' && \
      agent-deck session send '<Title>' '<prompt>'

    2. Poll until done:
    while true; do
      STATUS=$(agent-deck session show '<Title>' | grep '^Status:' | awk '{print $2}')
      if [ \"$STATUS\" = \"◐\" ]; then break; fi
      sleep 5
    done

    3. Get response:
    agent-deck session output '<Title>'

    Return the FULL response."
)
```

Then retrieve with `TaskOutput(task_id: "<id>", block: true)`.

**NEVER:** Poll in main conversation | Stop after send | Show intermediate checks

### Workflow 3: Sub-Agent (Child Session)

```bash
PARENT=$(agent-deck session current -q)
PROFILE=$(agent-deck session current --json | jq -r '.profile')

mkdir -p /tmp/subtask && \
  agent-deck -p "$PROFILE" add -t "Sub Task" --parent "$PARENT" -c claude /tmp/subtask && \
  agent-deck -p "$PROFILE" session start "Sub Task" && \
  agent-deck -p "$PROFILE" session send "Sub Task" "Sub-task prompt"
```

**Get sub-agent response:**
```bash
agent-deck session output "Sub Task"
```

### Workflow 4: Fork Claude Session

```bash
agent-deck session fork my-project -t "experiment" -g experiments
agent-deck session start experiment
```

## MCP Management

**Default: Do NOT attach MCPs** unless user explicitly requests.

```bash
# List available MCPs first
agent-deck mcp list

# Attach when creating (only if requested)
agent-deck add -t "Task" -c claude --mcp <mcp-name> /path

# Attach to existing session
agent-deck mcp attach <session> <mcp-name>
agent-deck session restart <session>  # Required to load
```

See [mcp-management.md](references/mcp-management.md) for full guide.

## Key Commands

### Session Lifecycle

```bash
agent-deck add [path] -t "title" -g "group" -c claude [--mcp name]
agent-deck session start <session>
agent-deck session stop <session>
agent-deck session restart <session>  # Reloads MCPs
agent-deck session attach <session>   # Ctrl+Q to detach
agent-deck session send <session> "message"
agent-deck session output <session>   # Get last response
agent-deck session current [-q|--json]  # Auto-detect current
```

### Groups & Profiles

```bash
agent-deck group create <name> [--parent <parent>]
agent-deck group move <session> <group>
agent-deck -p <profile> <command>  # Use specific profile
```

See [cli-reference.md](references/cli-reference.md) for complete command reference.
See [profiles.md](references/profiles.md) for profile management.

## Critical Notes

### Flag Ordering (Most Common Mistake)

**Flags MUST come BEFORE positional arguments:**

```bash
# Correct
agent-deck session start -m "Hello" "My Project"
agent-deck session show -json my-project

# WRONG (flag ignored!)
agent-deck session start "My Project" -m "Hello"
```

### Session Resolution

Commands accept flexible identifiers:
- **Title:** `"My Project"`
- **ID prefix:** `abc123` (>=6 chars)
- **Path:** `/Users/me/project`
- **Auto-detect:** Omit ID when inside tmux session

### MCP Decision Flow

| User Says | Action |
|-----------|--------|
| "Create a session" | NO MCPs |
| "Create session with X MCP" | `mcp list` -> verify -> attach |
| "Attach relevant MCPs" | `mcp list` -> analyze task -> choose -> attach |

## Reference Files

- [cli-reference.md](references/cli-reference.md) - Complete command reference
- [mcp-management.md](references/mcp-management.md) - MCP configuration guide
- [profiles.md](references/profiles.md) - Profile management
- [automation-patterns.md](references/automation-patterns.md) - Scripting patterns

## Troubleshooting

**Session shows "error":** Start with `agent-deck session start <session>`

**MCPs not loading:** Run `agent-deck session restart <session>`

**CLI changes not visible in TUI:** Press `Ctrl+R` in TUI to refresh

**Flag not working:** Ensure flag comes BEFORE positional arguments

See [cli-reference.md](references/cli-reference.md) for more troubleshooting.
