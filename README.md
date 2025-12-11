<div align="center">

```
┌──┬──┬──┐
│● │◐ │○ │
└──┴──┴──┘
```

# Agent Deck

**Terminal session manager for AI agents**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey)](https://github.com/asheshgoplani/agent-deck)

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Documentation](#documentation) • [Contributing](#contributing)

</div>

---

<!-- TODO: Add demo video here - drag mp4 into GitHub issue, copy URL -->
<!-- https://github.com/user-attachments/assets/YOUR-VIDEO-UUID -->

```
┌─────────────────────────────────────────────────────────────────────────┐
│  [●│◐│○] Agent Deck                          8 sessions   [/] Search    │
├────────────────────────┬────────────────────────────────────────────────┤
│  SESSIONS              │  Preview: devops/claude-1                      │
│  ▼ projects (4)     ●  │  $ claude                                      │
│    ▶ devops/claude  ●  │  I'll help you with the deployment...          │
│      devops/shell   ○  │                                                │
│      frontend       ◌  │  ┌─────────────────────────────────────────┐   │
│  ▼ personal (2)        │  │ Proceed with changes? (Y/n)             │   │
│      blog           ◌  │  └─────────────────────────────────────────┘   │
├────────────────────────┴────────────────────────────────────────────────┤
│ [↑↓] Navigate [Enter] Attach [/] Search [n] New [Tab] Fold [d] Del [q]  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Why Agent Deck?

Running multiple AI coding agents across projects gets messy fast. Agent Deck gives you a unified dashboard to manage all your sessions—Claude Code, Gemini CLI, Aider, Codex, or any terminal tool.

- **🔌 Universal** — Works with any terminal program, not locked to one AI
- **⚡ Fast** — Instant session creation, no forced program startup
- **📁 Organized** — Project-based hierarchy with collapsible groups
- **🔍 Searchable** — Find any session instantly with fuzzy search
- **🎯 Smart Status** — Knows when your agent is busy vs. waiting for input
- **🪨 Rock Solid** — Built on tmux, battle-tested for 20+ years

## Features

### 🚀 Claude Code Deep Integration

Agent Deck offers **first-class Claude Code integration** with powerful session forking:

```
┌─────────────────────────────────────────────────────────────┐
│  Parent Session                    │   Forked Sessions      │
│  ┌─────────────────┐               │   ┌─────────────────┐  │
│  │ "Build auth"    │──── Fork ────►│   │ "Try JWT"       │  │
│  │ claude session  │               │   └─────────────────┘  │
│  │                 │──── Fork ────►│   ┌─────────────────┐  │
│  │                 │               │   │ "Try OAuth"     │  │
│  └─────────────────┘               │   └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Fork a conversation** to explore multiple approaches in parallel:
- Press `f` to quick-fork any Claude session
- Press `F` to fork with custom name/group
- Each fork **inherits full conversation context** from parent
- Forks get their own session ID—can be forked again!

**Use cases:**
- 🔀 **Branching explorations** — Try different implementation approaches from the same context
- 🧪 **Experiment safely** — Fork before risky changes, keep original intact
- 👥 **Parallel work** — Multiple Claude instances working from same knowledge base
- 📚 **Learning** — Fork to ask "what if" questions without derailing main session

**Automatic session detection:**
- Detects Claude session ID from `.jsonl` files
- Tracks sessions across restarts
- Handles multiple Claude sessions in same project
- Works with custom Claude profiles (`CLAUDE_CONFIG_DIR`)

### Intelligent Status Detection

Agent Deck automatically detects what your AI agent is doing:

| Status | Symbol | Meaning |
|--------|--------|---------|
| **Running** | `●` green | Agent is actively working |
| **Waiting** | `◐` yellow | Prompt detected, needs your input |
| **Idle** | `○` gray | Session ready, nothing happening |
| **Error** | `✕` red | Session has an error |

Works out-of-the-box with Claude Code, Gemini CLI, Aider, and Codex—detecting busy indicators, permission prompts, and input requests.

### Supported Tools

| Icon | Tool | Status Detection |
|------|------|------------------|
| 🤖 | Claude Code | Busy indicators, permission dialogs, prompts |
| ✨ | Gemini CLI | Activity detection, prompts |
| 🔧 | Aider | Y/N prompts, input detection |
| 💻 | Codex | Prompts, continuation requests |
| 🐚 | Any Shell | Standard shell prompts |

## Installation

### Prerequisites

- **macOS**, **Linux**, or **Windows (via WSL)**
- **[tmux](https://github.com/tmux/tmux)** — Terminal multiplexer
  ```bash
  # macOS
  brew install tmux

  # Ubuntu/Debian/WSL
  sudo apt install tmux

  # Fedora
  sudo dnf install tmux
  ```

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
```

This downloads the latest release and installs to `~/.local/bin`.

**Options:**
```bash
# Custom binary name
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash -s -- --name ad

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash -s -- --dir /usr/local/bin

# Specific version
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash -s -- --version v0.2.0
```

### Homebrew (macOS/Linux)

```bash
brew install asheshgoplani/tap/agent-deck
```

### Go Install

```bash
go install github.com/asheshgoplani/agent-deck/cmd/agent-deck@latest
```

Requires Go 1.24+ and `$GOPATH/bin` in your PATH.

### From Source

```bash
git clone https://github.com/asheshgoplani/agent-deck.git
cd agent-deck
make install          # Install to /usr/local/bin (requires sudo)
# or
make install-user     # Install to ~/.local/bin (no sudo)
```

### Verify Installation

```bash
agent-deck version
```

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
| `R` | Rename session/group |
| `m` | Move session to group |
| `d` | Delete |
| `K` / `J` | Reorder up/down |

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
| `i` | Import existing tmux sessions |
| `r` | Refresh |

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

For the best experience, add these settings to your `~/.tmux.conf`:

```bash
# Enable mouse scrolling in attached sessions
set -g mouse on

# Increase scrollback buffer (AI agents produce lots of output)
set -g history-limit 50000

# Vi-style navigation in copy mode (Prefix + [ to enter, k/j to scroll)
setw -g mode-keys vi
```

After editing, reload your config:
```bash
tmux source-file ~/.tmux.conf
```

**Why these settings matter:**
| Setting | Without it |
|---------|------------|
| `mouse on` | Mouse wheel scrolling won't work |
| `history-limit` | Older AI output gets truncated (default is only 2000 lines) |
| `mode-keys vi` | No keyboard scrolling in copy mode |

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
