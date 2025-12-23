<div align="center">

<!-- Status Grid Logo -->
<img src="site/logo.svg" alt="Agent Deck Logo" width="120">

# Agent Deck

**Terminal session manager for AI agents**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey)](https://github.com/asheshgoplani/agent-deck)

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [CLI Commands](#cli-commands) • [Documentation](#documentation)

</div>

---

![Agent Deck Demo](demos/agent-deck-overview.gif)

## Why Agent Deck?

Managing multiple AI coding sessions across projects can get overwhelming. Agent Deck provides a single dashboard to monitor and switch between all your sessions: Claude Code, Gemini CLI, Aider, Codex, or any terminal tool.

**What it does:**
- Organize sessions by project with collapsible groups
- See at a glance which agents are running, waiting, or idle
- Switch between sessions instantly with keyboard shortcuts
- Search and filter to find what you need
- Built on tmux for reliability

## Features

### 🚀 Session Forking (Claude Code)

Fork Claude conversations to explore multiple approaches in parallel. Each fork inherits full conversation context.

![Fork Session Demo](demos/fork-session.gif)

- Press `f` to quick-fork, `F` for custom name/group
- Forks inherit context and can be forked again
- Auto-detects Claude session ID across restarts

### 🔌 MCP Manager

Attach and detach MCP servers on the fly. No config editing required.

![MCP Manager Demo](demos/mcp-manager.gif)

- Press `M` to open, `Space` to toggle MCPs
- **LOCAL** scope (project) or **GLOBAL** (all projects)
- Session auto-restarts with new MCPs loaded

**Adding Available MCPs:**

Define your MCPs once in `~/.agent-deck/config.toml`, then toggle them per project:

```toml
# Web search
[mcps.exa]
command = "npx"
args = ["-y", "exa-mcp-server"]
env = { EXA_API_KEY = "your-api-key" }
description = "Web search via Exa AI"

# GitHub integration
[mcps.github]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { GITHUB_PERSONAL_ACCESS_TOKEN = "ghp_your_token" }
description = "GitHub repos, issues, PRs"

# Browser automation
[mcps.playwright]
command = "npx"
args = ["-y", "@playwright/mcp@latest"]
description = "Browser automation & testing"

# Memory across sessions
[mcps.memory]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-memory"]
description = "Persistent memory via knowledge graph"
```

<details>
<summary>More MCP examples</summary>

```toml
# YouTube transcripts
[mcps.youtube-transcript]
command = "npx"
args = ["-y", "@kimtaeyoon83/mcp-server-youtube-transcript"]
description = "Get YouTube transcripts"

# Web scraping
[mcps.firecrawl]
command = "npx"
args = ["-y", "firecrawl-mcp"]
env = { FIRECRAWL_API_KEY = "your-key" }
description = "Web scraping and crawling"

# Notion
[mcps.notion]
command = "npx"
args = ["-y", "@notionhq/notion-mcp-server"]
env = { NOTION_TOKEN = "your-token" }
description = "Notion workspace access"

# Sequential thinking
[mcps.sequential-thinking]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-sequential-thinking"]
description = "Step-by-step reasoning"

# Context7 - code docs
[mcps.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp@latest"]
description = "Up-to-date code documentation"

# Anthropic docs
[mcps.anthropic-docs]
command = "npx"
args = ["-y", "anthropic-docs-mcp", "--transport", "stdio"]
description = "Search Claude & Anthropic docs"
```

</details>

### 🔍 Search

Press `/` to search across sessions with fuzzy matching. Filter by status with `!` (running), `@` (waiting), `#` (idle), `$` (error).

### 🎯 Smart Status Detection

Automatically detects what your AI agent is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| **Running** | `●` green | Agent is working |
| **Waiting** | `◐` yellow | Needs input |
| **Idle** | `○` gray | Ready |
| **Error** | `✕` red | Error |

Works with Claude Code, Gemini CLI, Aider, Codex, and any shell.

## Installation

**Works on:** macOS • Linux • Windows (WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
```

The installer downloads the binary, installs tmux if needed, and configures tmux for mouse/clipboard support.

Then run: `agent-deck`

> **Windows:** [Install WSL](https://learn.microsoft.com/en-us/windows/wsl/install) first.

<details>
<summary>Other install methods</summary>

**Homebrew**
```bash
brew install asheshgoplani/tap/agent-deck
```

**Go**
```bash
go install github.com/asheshgoplani/agent-deck/cmd/agent-deck@latest
```

**From Source**
```bash
git clone https://github.com/asheshgoplani/agent-deck.git && cd agent-deck && make install
```

</details>

## Usage

```bash
agent-deck                    # Launch TUI
agent-deck add .              # Add current directory as session
agent-deck add . -c claude    # Add with Claude Code
agent-deck list               # List all sessions
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `Enter` | Attach to session |
| `n` | New session |
| `g` | New group |
| `r` | Rename |
| `d` | Delete |
| `f` | Fork Claude session |
| `M` | MCP Manager |
| `/` | Search |
| `Ctrl+Q` | Detach from session |
| `?` | Help |

## CLI Commands

Agent Deck provides a full CLI for automation and scripting. All commands support `--json` for machine-readable output and `-p, --profile` for profile selection.

> **Note:** Flags must come BEFORE positional arguments (Go flag package standard).

### Quick Reference

```bash
agent-deck                              # Launch TUI
agent-deck add . -c claude              # Add session with Claude
agent-deck list --json                  # List sessions as JSON
agent-deck status                       # Quick status overview
agent-deck session attach my-project    # Attach to session
```

### Session Commands

Manage individual sessions. Sessions can be identified by:
- **Title**: `my-project` (exact or partial match)
- **ID prefix**: `a1b2c3` (first 6+ chars)
- **Path**: `/Users/me/project`

```bash
# Start/Stop/Restart
agent-deck session start <id>           # Start session's tmux process
agent-deck session stop <id>            # Stop/kill session process
agent-deck session restart <id>         # Restart (Claude: reloads MCPs)

# Fork (Claude only)
agent-deck session fork <id>            # Fork with inherited context
agent-deck session fork <id> -t "exploration"       # Custom title
agent-deck session fork <id> -g "experiments"       # Into specific group

# Attach/Show
agent-deck session attach <id>          # Attach interactively
agent-deck session show <id>            # Show session details
agent-deck session show                 # Auto-detect current session (in tmux)
```

**Fork flags:**
| Flag | Description |
|------|-------------|
| `-t, --title` | Custom title for forked session |
| `-g, --group` | Target group for forked session |

### MCP Commands

Manage Model Context Protocol servers for Claude sessions.

```bash
# List available MCPs (from config.toml)
agent-deck mcp list
agent-deck mcp list --json

# Show attached MCPs for a session
agent-deck mcp attached <id>
agent-deck mcp attached                 # Auto-detect current session

# Attach/Detach MCPs
agent-deck mcp attach <id> github       # Attach to LOCAL scope
agent-deck mcp attach <id> exa --global # Attach to GLOBAL scope
agent-deck mcp attach <id> memory --restart  # Attach and restart session

agent-deck mcp detach <id> github       # Detach from LOCAL
agent-deck mcp detach <id> exa --global # Detach from GLOBAL
```

**MCP flags:**
| Flag | Description |
|------|-------------|
| `--global` | Apply to global Claude config (all projects) |
| `--restart` | Restart session after change (loads new MCPs) |

### Group Commands

Organize sessions into hierarchical groups.

```bash
# List groups
agent-deck group list
agent-deck group list --json

# Create groups
agent-deck group create work            # Create root group
agent-deck group create frontend --parent work  # Create subgroup

# Delete groups
agent-deck group delete old-projects    # Delete (fails if has sessions)
agent-deck group delete old-projects --force    # Move sessions to default, then delete

# Move sessions
agent-deck group move my-session work   # Move session to group
```

**Group flags:**
| Flag | Description |
|------|-------------|
| `--parent` | Parent group for creating subgroups |
| `--force` | Force delete by moving sessions to default group |

### Status Command

Quick status check without launching the TUI.

```bash
agent-deck status                       # Compact: "2 waiting - 5 running - 3 idle"
agent-deck status -v                    # Verbose: detailed list by status
agent-deck status -q                    # Quiet: just waiting count (for prompts)
agent-deck status --json                # JSON output
```

### Global Flags

These flags work with all commands:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (for automation) |
| `-q, --quiet` | Minimal output, rely on exit codes |
| `-p, --profile <name>` | Use specific profile |

### Examples

**Scripting with JSON output:**
```bash
# Get all running sessions
agent-deck list --json | jq '.[] | select(.status == "running")'

# Count waiting sessions
agent-deck status -q  # Returns just the number

# Check if specific session exists
agent-deck session show my-project --json 2>/dev/null && echo "exists"
```

**Automation workflows:**
```bash
# Start all sessions in a group
agent-deck list --json | jq -r '.[] | select(.group == "work") | .id' | \
  xargs -I{} agent-deck session start {}

# Attach MCP to all Claude sessions
agent-deck list --json | jq -r '.[] | select(.tool == "claude") | .id' | \
  xargs -I{} agent-deck mcp attach {} memory --restart
```

**Current session detection (inside tmux):**
```bash
# Show current session info
agent-deck session show

# Show MCPs for current session
agent-deck mcp attached
```

## Documentation

### Project Organization

```
▼ Projects (3)
  ├─ frontend     ●
  ├─ backend      ◐
  └─ api          ○
▼ Personal
  └─ blog         ○
```

Sessions are organized in collapsible groups. Create nested groups, reorder items, and import existing tmux sessions with `i`.

### Configuration

Data stored in `~/.agent-deck/`:

```
~/.agent-deck/
├── sessions.json     # Sessions and groups
└── config.toml       # User config (optional)
```

For custom Claude profile directory:

```toml
[claude]
config_dir = "~/.claude-work"
```

### tmux Configuration

The installer configures tmux automatically. For manual setup, see the [tmux configuration guide](https://github.com/asheshgoplani/agent-deck/wiki/tmux-Configuration).

## Development

```bash
make build    # Build
make test     # Test
make lint     # Lint
```

## Contributing

Contributions welcome! Fork, create a branch, and open a PR.

## License

MIT License - see [LICENSE](LICENSE)

---

<div align="center">

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [tmux](https://github.com/tmux/tmux)

</div>
