<div align="center">

<!-- Status Grid Logo -->
<img src="site/logo.svg" alt="Agent Deck Logo" width="120">

# Agent Deck

**Terminal session manager for AI agents**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey)](https://github.com/asheshgoplani/agent-deck)

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Documentation](#documentation) • [Contributing](#contributing)

</div>

---

![Agent Deck Demo](demos/agent-deck-overview.gif)

## Why Agent Deck?

Running multiple AI coding agents across projects gets messy fast. Agent Deck gives you a unified dashboard to manage all your sessions—Claude Code, Gemini CLI, Aider, Codex, or any terminal tool.

- **🔌 Universal** — Works with any terminal program, not locked to one AI
- **⚡ Fast** — Instant session creation, no forced program startup
- **📁 Organized** — Project-based hierarchy with collapsible groups
- **🔍 Searchable** — Find any session instantly with fuzzy search
- **🎯 Smart Status** — Knows when your agent is busy vs. waiting for input
- **🪨 Rock Solid** — Built on tmux, battle-tested for 20+ years

## Features

### 🚀 Session Forking (Claude Code)

Fork Claude conversations to explore multiple approaches in parallel. Each fork inherits full conversation context.

![Fork Session Demo](demos/fork-session.gif)

- Press `f` to quick-fork, `F` for custom name/group
- Forks inherit context and can be forked again
- Auto-detects Claude session ID across restarts

### 🔌 MCP Manager

Attach and detach MCP servers on the fly—no config editing required.

![MCP Manager Demo](demos/mcp-manager.gif)

- Press `M` to open, `Space` to toggle MCPs
- **LOCAL** scope (project) or **GLOBAL** (all projects)
- Session auto-restarts with new MCPs loaded

### 🔍 Global Search

Press `G` to search across **all your Claude conversations**—full content search with keyword highlighting and smart ranking.

### 🎯 Smart Status Detection

Automatically detects what your AI agent is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| **Running** | `●` green | Agent is actively working |
| **Waiting** | `◐` yellow | Needs your input |
| **Idle** | `○` gray | Session ready |
| **Error** | `✕` red | Session error |

Works with Claude Code, Gemini CLI, Aider, and Codex out-of-the-box.

### ⌨️ Quick Filters

Filter by status with `!` (running), `@` (waiting), `#` (idle), `$` (error), or `0` (all).

### Supported Tools

Each AI tool displays with its brand color in the session list for easy visual identification:

| Icon | Tool | Badge Color | Status Detection |
|------|------|-------------|------------------|
| 🤖 | Claude Code | Orange | Busy indicators, permission dialogs, prompts |
| ✨ | Gemini CLI | Purple | Activity detection, prompts |
| 🔧 | Aider | Red | Y/N prompts, input detection |
| 💻 | Codex | Cyan | Prompts, continuation requests |
| 🖱️ | Cursor | Blue | Activity detection |
| 🐚 | Any Shell | Default | Standard shell prompts |

## Installation

**Works on:** macOS • Linux • Windows (WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
```

**The installer handles everything:**
1. Downloads and installs the binary
2. Installs tmux if not present
3. **Configures tmux** for mouse scrolling & clipboard (you'll be prompted)

Then run: `agent-deck`

> **Windows:** [Install WSL](https://learn.microsoft.com/en-us/windows/wsl/install) first, then run the command above.

<details>
<summary>Installer options</summary>

```bash
# Custom install location
curl -fsSL .../install.sh | bash -s -- --dir /usr/local/bin

# Specific version
curl -fsSL .../install.sh | bash -s -- --version v0.5.7

# Skip tmux configuration only
curl -fsSL .../install.sh | bash -s -- --skip-tmux-config

# Fully automated (no prompts - for CI/scripts)
curl -fsSL .../install.sh | bash -s -- --non-interactive
```

</details>

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

### Launch the TUI

```bash
agent-deck
```

### CLI Commands

```bash
# Add a session
agent-deck add .                              # Current directory
agent-deck add ~/projects/myapp               # Specific path
agent-deck add . -t "My App" -g work          # With title and group
agent-deck add . -c claude                    # With command (claude, gemini, aider, codex)

# List sessions
agent-deck list                               # Table format
agent-deck list --json                        # JSON for scripting

# Remove a session
agent-deck remove <id|title>                  # By ID or title
```

### Keyboard Shortcuts

#### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Collapse group |
| `l` / `→` / `Tab` | Expand group |
| `Enter` | Attach to session |

#### Session Management
| Key | Action |
|-----|--------|
| `n` | New session |
| `g` | New group |
| `r` | Rename session/group |
| `Shift+R` | Restart session |
| `m` | Move session to group |
| `d` | Delete |
| `u` | Mark unread |
| `K` / `J` | Reorder up/down |
| `Shift+M` | MCP Manager (Claude only) |

#### Claude Code Integration
| Key | Action |
|-----|--------|
| `f` | Quick fork Claude session (inherits conversation context) |
| `F` | Fork with custom name/group |

*Fork requires an active Claude Code session with a valid session ID.*

#### Search & Import
| Key | Action |
|-----|--------|
| `/` | Search sessions |
| `G` | Global Search (all Claude conversations) |
| `i` | Import existing tmux sessions |
| `?` | Help (keyboard shortcuts) |

#### Quick Filters
| Key | Action |
|-----|--------|
| `0` | Show all sessions (clear filter) |
| `!` | Filter to running sessions only |
| `@` | Filter to waiting sessions only |
| `#` | Filter to idle sessions only |
| `$` | Filter to error sessions only |

#### While Attached
| Key | Action |
|-----|--------|
| `Ctrl+Q` | Detach (session keeps running) |

## Documentation

### Project Organization

Sessions are organized in a hierarchical folder structure:

```
▼ Projects (5)
  ├─ frontend          ●
  ├─ backend           ◐
  └─ ▼ devops (2)
       ├─ deploy       ○
       └─ monitor      ○
▼ Personal (2)
  └─ blog              ○
```

- Groups can be nested to any depth
- Sessions inherit their parent group
- Empty groups persist until deleted
- Order is preserved and customizable

### Session Preview

The preview pane shows:
- Live terminal output (last lines)
- Session metadata (path, tool, group)
- Current status

### Import Existing Sessions

Press `i` to discover tmux sessions not created by Agent Deck. It will:
1. Find all tmux sessions
2. Auto-detect the tool from session name
3. Auto-group by project directory
4. Add to Agent Deck for unified management

### Configuration

Data is stored in `~/.agent-deck/`:

```
~/.agent-deck/
├── sessions.json     # Sessions, groups, state
├── config.toml       # User configuration (optional)
└── hooks/            # Hook scripts (optional)
```

### Recommended tmux Configuration

> **Note:** The install script offers to configure tmux automatically. The configs below are for reference or manual setup.

For optimal experience with mouse copy, scroll, and clipboard integration, use this config in `~/.tmux.conf`:

<details>
<summary><strong>macOS</strong></summary>

```bash
# ============================================
# Minimal tmux Configuration for macOS
# Mouse copy, scroll, and clipboard - just works
# ============================================

# ----- Terminal -----
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",*256col*:Tc"

# ----- Performance -----
set -sg escape-time 0
set -g history-limit 10000

# ----- Mouse (enables scroll + drag-to-copy) -----
set -g mouse on

# ----- Clipboard -----
set -s set-clipboard external

# Mouse drag automatically copies to system clipboard
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "pbcopy"

# Double-click selects word, triple-click selects line (auto-copies)
bind-key -T copy-mode-vi DoubleClick1Pane select-pane \; send-keys -X select-word \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode DoubleClick1Pane select-pane \; send-keys -X select-word \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode-vi TripleClick1Pane select-pane \; send-keys -X select-line \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "pbcopy"
bind-key -T copy-mode TripleClick1Pane select-pane \; send-keys -X select-line \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "pbcopy"
```

</details>

<details>
<summary><strong>Linux (X11 with xclip)</strong></summary>

```bash
# ============================================
# Minimal tmux Configuration for Linux
# Mouse copy, scroll, and clipboard - just works
# ============================================

# ----- Terminal -----
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",*256col*:Tc"

# ----- Performance -----
set -sg escape-time 0
set -g history-limit 10000

# ----- Mouse (enables scroll + drag-to-copy) -----
set -g mouse on

# ----- Clipboard -----
set -s set-clipboard external

# Mouse drag automatically copies to system clipboard
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"
bind-key -T copy-mode MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"

# Double-click selects word, triple-click selects line (auto-copies)
bind-key -T copy-mode-vi DoubleClick1Pane select-pane \; send-keys -X select-word \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"
bind-key -T copy-mode DoubleClick1Pane select-pane \; send-keys -X select-word \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"
bind-key -T copy-mode-vi TripleClick1Pane select-pane \; send-keys -X select-line \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"
bind-key -T copy-mode TripleClick1Pane select-pane \; send-keys -X select-line \; run-shell -d 0.3 \; send-keys -X copy-pipe-and-cancel "xclip -in -selection clipboard"
```

For Wayland, replace `xclip -in -selection clipboard` with `wl-copy`.

</details>

After adding, reload:
```bash
tmux source-file ~/.tmux.conf
```

**What this config does:**

| Feature | How it works |
|---------|--------------|
| **Drag to copy** | Click and drag → auto-copies to clipboard |
| **Double-click** | Selects word → auto-copies |
| **Triple-click** | Selects line → auto-copies |
| **Mouse scroll** | Just scroll with mouse wheel |
| **Paste** | `Cmd+V` (macOS) or `Ctrl+Shift+V` (Linux) |

**Why these settings:**

| Setting | Purpose |
|---------|---------|
| `escape-time 0` | No delay on ESC key (fixes sluggishness) |
| `history-limit 10000` | AI agents produce lots of output (default is 2000) |
| `set-clipboard external` | Secure clipboard (apps inside tmux can't hijack it) |
| `MouseDragEnd1Pane` | Auto-copy on mouse release |

> **Tip:** Hold **Shift** while selecting to bypass tmux and use native terminal selection.

### Claude Code Profile (Optional)

If you use a custom Claude profile directory (e.g., dual account setup), configure it in `~/.agent-deck/config.toml`:

```toml
[claude]
config_dir = "~/.claude-work"
```

This tells Agent Deck where to find Claude session data for:
- Session ID detection
- Fork functionality
- Session tracking across restarts

### Hook Integration (Optional)

For instant status updates without polling, configure hooks in your AI tool:

**Claude Code** (`~/.claude/settings.json`):
```json
{
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "~/.agent-deck/hooks/claude-code.sh"}]}]
  }
}
```

## Development

```bash
make build      # Build binary
make test       # Run tests
make dev        # Run with auto-reload (requires 'air')
make fmt        # Format code
make lint       # Lint code (requires 'golangci-lint')
make release    # Cross-platform builds
make clean      # Clean build artifacts
```

### Project Structure

```
agent-deck/
├── cmd/agent-deck/        # CLI entry point
├── internal/
│   ├── ui/                # TUI components (Bubble Tea)
│   ├── session/           # Session & group management
│   └── tmux/              # tmux integration, status detection
├── Makefile
├── go.mod
└── README.md
```

### Debug Mode

```bash
AGENTDECK_DEBUG=1 agent-deck
```

Logs status transitions to stderr for troubleshooting.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Comparison

| Feature | Agent Deck | Alternatives |
|---------|------------|--------------|
| Universal (any tool) | ✅ | Often tool-specific |
| **Claude Code fork** | ✅ Context inheritance | ❌ Not available |
| Fast session creation | ✅ Instant | Slow startup |
| Project hierarchy | ✅ Nested groups | Flat lists |
| Session search | ✅ Fuzzy search | Limited |
| Import existing | ✅ tmux discovery | Manual only |
| Smart status | ✅ Per-tool detection | Basic |
| Memory footprint | ~20MB | Higher |

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Style definitions
- [tmux](https://github.com/tmux/tmux) — Terminal multiplexer

---

<div align="center">

**[⬆ Back to Top](#agent-deck)**

</div>
