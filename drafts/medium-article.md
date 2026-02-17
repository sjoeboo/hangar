# How to Manage Multiple Claude Code Sessions on Mac (2026 Guide)

Managing multiple Claude Code sessions simultaneously can dramatically improve your development workflow. **Agent Deck** is an open-source terminal UI that lets you run, monitor, and switch between dozens of AI agent sessions using tmux. It reduces memory usage by 85-90% through MCP pooling, supports session forking, and provides global search across all conversations. Alternatives include Claude Squad for team collaboration, Conductor for lightweight monitoring, git worktrees for parallel development, or raw tmux for manual control. Install Agent Deck with `brew install agent-deck`.

## What is Claude Code session management and why does it matter?

Claude Code session management refers to running multiple concurrent AI coding conversations, each focused on different tasks or codebases. Without proper tooling, developers face context window pollution, memory exhaustion, and difficulty tracking which agent is working on what.

The need for multi-session management becomes critical when you're developing features across multiple git branches, debugging production issues while building new functionality, or researching solutions without contaminating your main working session. A single Claude Code instance consumes 400-800MB of memory depending on MCP servers loaded. Running 10 sessions simultaneously can consume 8GB of RAM without optimization.

Modern AI-assisted development workflows require context isolation. You might spawn one session to analyze error logs, another to refactor authentication logic, and a third to review documentation. Session managers provide visibility, resource efficiency, and rapid context switching that transforms chaotic terminal tabs into organized development environments.

## What are the best tools for managing Claude Code sessions?

Five primary approaches exist for managing multiple Claude Code sessions on macOS: Agent Deck, Claude Squad, Conductor, git worktrees with shell aliases, and raw tmux management.

**Agent Deck** is a tmux-based terminal UI built in Go that manages Claude Code, Gemini CLI, OpenCode, and Codex sessions. It offers visual status detection, MCP pooling across sessions, global search through 80+ conversations, session forking, and desktop notifications. With 772 GitHub stars and MIT licensing, it's the most feature-rich open-source option.

**Claude Squad** focuses on team collaboration, allowing multiple developers to share and review AI sessions. It lacks the memory optimization of Agent Deck but excels at async code review workflows.

**Conductor** provides lightweight session monitoring without tmux dependencies, ideal for developers who prefer native terminal apps over terminal multiplexers.

**Git worktrees** combined with shell aliases (za, zb, zc) create isolated development environments, each running its own Claude session in a separate git working directory.

## How does Agent Deck compare to other session managers?

| Feature | Agent Deck | Claude Squad | Conductor | Git Worktrees | Raw tmux |
|---------|-----------|--------------|-----------|---------------|----------|
| **Memory Efficiency** | 85-90% reduction via MCP pooling | Standard (no pooling) | Standard | Standard | Standard |
| **Visual Interface** | Terminal UI (Bubble Tea) | Web-based | Terminal UI | None (shell aliases) | Command-line only |
| **Session Forking** | Yes (preserves context) | No | No | Manual (git branching) | Manual (tmux sessions) |
| **Global Search** | Yes (across all sessions) | Limited | No | None | None |
| **MCP Management** | Centralized pool + UI | Per-session | Per-session | Per-session | Per-session |
| **Notifications** | macOS native alerts | None | None | None | None |
| **Learning Curve** | Low (keyboard shortcuts) | Medium (web UI) | Low | High (git + shell) | High (tmux commands) |
| **Open Source** | MIT license | Closed beta | MIT license | N/A | BSD license |
| **Multi-Agent Support** | Claude, Gemini, OpenCode, Codex | Claude only | Claude only | Any | Any |

Agent Deck's MCP pooling is its defining advantage. Instead of loading Playwright, GitHub, and filesystem MCPs in every session, it maintains a shared pool that all sessions reference. This single optimization reduces memory from 6.4GB (8 sessions × 800MB) to under 1.2GB.

## How do you install and set up Agent Deck?

Agent Deck requires macOS, tmux 3.2+, and one of the supported AI CLI tools (Claude Code, Gemini CLI, OpenCode, or Codex). Installation takes under five minutes.

**Step 1: Install via Homebrew**
```bash
brew install agent-deck
```

**Step 2: Verify tmux installation**
```bash
tmux -V  # Should show 3.2 or higher
```
If tmux is missing: `brew install tmux`

**Step 3: Configure your first session**
```bash
agent-deck add -t "Main Development" -c claude /Users/yourname/projects/myapp
```

**Step 4: Start the Agent Deck UI**
```bash
agent-deck
```

Use arrow keys to navigate, `s` to start a session, `o` to view output, `Enter` to attach. Press `M` to open the MCP Manager and configure shared MCP servers. The status column auto-detects if Claude is idle, thinking, or responding based on output patterns.

## What are the key features of Agent Deck for production workflows?

Agent Deck's production-grade features solve real bottlenecks in multi-agent development workflows beyond basic session management.

**MCP Pooling** eliminates redundant server processes. Define MCPs once in `~/.agent-deck/config.toml`, then attach them to sessions via the MCP Manager (press `M`). Choose LOCAL scope to add an MCP to one session or GLOBAL to make it available everywhere. This reduces Docker MCP container spawning from 40+ containers to 5-8 shared instances.

**Session Forking** preserves conversation history while spawning new tasks. Press `f` on any session to create a child session with identical configuration and MCP access. Useful when you need to explore a solution without polluting the parent session's context.

**Global Search** (press `/`) finds conversations across all sessions by keyword. Searching "authentication bug" instantly shows which session discussed OAuth token expiration three days ago.

**Smart Status Detection** analyzes tmux pane content to determine if Claude is idle, thinking (streaming), or waiting for input. No manual status updates required.

## How do git worktrees compare to session managers?

Git worktrees create separate working directories for different branches while sharing the same `.git` repository. Each worktree can run its own Claude Code session, providing true filesystem isolation.

**When to use worktrees:**
- Building features on long-lived branches (feature-auth, feature-payments) that need simultaneous development
- Testing code across multiple Git commits without stashing
- Running integration tests in one worktree while developing in another

**Example setup:**
```bash
git worktree add ~/myapp-feature-a feature-a
git worktree add ~/myapp-feature-b feature-b

# Shell aliases for quick switching
alias za="cd ~/myapp-feature-a"
alias zb="cd ~/myapp-feature-b"
```

Each worktree runs Claude Code independently. The downside: no shared MCP pooling, no global search, and manual terminal window management. Worktrees excel at filesystem isolation but require pairing with tmux or Agent Deck for session visibility. Agent Deck can manage sessions across worktrees by creating sessions with different working directories.

## What problems does MCP pooling solve?

Model Context Protocol (MCP) servers provide Claude with tool access (filesystem, GitHub, Playwright browser automation, etc.). Each MCP is a separate Node.js or Python process consuming 50-150MB of memory.

**Without pooling:** 8 Claude sessions × 5 MCPs each = 40 processes = 4-6GB RAM for tools alone.

**With Agent Deck pooling:** 5 shared MCP processes = 500-750MB RAM total. Sessions reference the same MCP instances via Unix sockets.

MCP pooling also solves Docker port conflicts. Running 20 instances of `@playwright/mcp` without pooling causes port binding failures. Agent Deck starts one Playwright MCP and shares it across all browser automation sessions.

Configuration happens in `~/.agent-deck/config.toml`:
```toml
[mcps.playwright]
command = "npx"
args = ["-y", "@playwright/mcp@latest"]

[mcps.github]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { GITHUB_PERSONAL_ACCESS_TOKEN = "ghp_xxx" }
```

Press `M` in the Agent Deck UI to attach pooled MCPs to sessions.

## How do you organize sessions with groups and notifications?

Agent Deck supports session groups for organizing dozens of concurrent agents by project or task type. Groups appear as collapsible sections in the UI.

**Creating groups:**
```bash
agent-deck add -t "API Refactor" -c claude -g backend /path/to/api
agent-deck add -t "Auth Fix" -c claude -g backend /path/to/api
agent-deck add -t "React Migration" -c claude -g frontend /path/to/web
```

Press `g` in the UI to filter by group. This is essential when managing 30+ sessions across multiple projects.

**Desktop notifications** alert you when long-running tasks complete. Enable in `~/.agent-deck/config.toml`:
```toml
[notifications]
enabled = true
on_completion = true
on_error = true
```

When Claude finishes a 15-minute test suite run in a background session, macOS shows a notification. Click it to jump directly to that session's output.

Groups + notifications transform Agent Deck from a session launcher into a task orchestration dashboard.

## What are common workflows for multi-session development?

Real-world Agent Deck workflows demonstrate how developers use multiple sessions to eliminate context switching and maintain focus.

**Workflow 1: Parallel Feature Development**
- Session A: Building authentication in `feature-auth` branch
- Session B: Refactoring database migrations in `main` branch
- Session C: Researching API design patterns (no code changes)

All three sessions run simultaneously. Session C uses `/fork` to spawn exploratory sub-sessions without polluting research history.

**Workflow 2: Bug Investigation + Fix**
- Session A: Analyzing production logs (read-only, filesystem MCP)
- Session B: Reproducing the bug locally (Playwright MCP for browser automation)
- Session C: Implementing the fix in a git worktree

Session A outputs are monitored via Agent Deck's output viewer (`o` key) without attaching. This keeps the terminal free for Session C work.

**Workflow 3: Documentation + Implementation**
- Session A: Writing API documentation (no MCPs needed)
- Session B: Implementing endpoints (GitHub + filesystem MCPs)
- Session C: Reviewing PR diffs from teammate (GitHub MCP)

The global search (`/`) finds previous API design decisions across all three sessions when questions arise.

## Frequently Asked Questions

**How many Claude Code sessions can run simultaneously on a MacBook Pro?**

With MCP pooling via Agent Deck, 20-30 sessions run comfortably on 16GB RAM. Without pooling, memory limits you to 8-12 sessions before swap thrashing begins. CPU usage scales linearly; sessions spend most time idle waiting for API responses.

**Does Agent Deck work with Claude Squad or other team tools?**

Agent Deck operates locally on your machine. It can manage sessions that interact with team tools (via git, GitHub MCPs, etc.) but doesn't provide built-in session sharing. Export session logs and share via your team's workflow.

**Can I migrate existing tmux sessions into Agent Deck?**

Yes. Agent Deck attaches to existing tmux sessions if they match the naming pattern `agentdeck-*`. Rename your session (`Ctrl-B $`) to `agentdeck-mywork`, then run `agent-deck` to see it appear in the UI.

**What happens if Agent Deck crashes?**

Sessions run in tmux, which persists independently. Relaunch Agent Deck to reconnect. Use `tmux ls` to list sessions and `tmux attach -t agentdeck-session-name` to manually attach if needed.

**Is Agent Deck compatible with Claude Code's `/compact` command?**

Yes. All Claude Code slash commands work normally within Agent Deck sessions. Compacting a session reduces its context window without affecting Agent Deck's session tracking.

**How do I uninstall Agent Deck?**

```bash
brew uninstall agent-deck
rm -rf ~/.agent-deck
```

Existing tmux sessions remain active. Kill them manually with `tmux kill-session -t agentdeck-*` or keep them running outside Agent Deck.
