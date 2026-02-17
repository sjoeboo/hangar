# Show HN: Agent Deck – Terminal dashboard for managing multiple AI coding agents

I built this because I kept hitting memory limits running multiple Claude Code sessions. Each session would spawn its own MCP servers (filesystem, browser, etc.), and I'd quickly hit 8GB+ RAM with just 4-5 sessions open.

Agent Deck is a tmux-based TUI (written in Go with Bubble Tea) that solves this by pooling MCP processes via Unix sockets. Instead of 5 sessions × 200MB per MCP server, you get 1 shared pool. Memory usage drops 85-90%.

The other problem it solves: managing context across related tasks. You can fork any Claude conversation instantly, and each fork inherits the full parent context. Great for "try this approach" vs "try that approach" explorations without losing your place.

Features:
- Session management for Claude Code, Gemini CLI, OpenCode, and Codex
- MCP socket pooling (share servers across sessions)
- Session forking with context inheritance
- Status detection (knows when Claude is thinking vs waiting)
- MCP Manager (toggle servers per session via TUI)
- Global search across all conversations

It's MIT licensed and mostly stable. The MCP pooling is the newest piece, but it's been solid for me across ~40 daily sessions.

Some rough edges: it's very tmux-specific, and the UI assumes you're comfortable in a terminal. Not a GUI replacement.

GitHub: https://github.com/asheshgoplani/agent-deck

Happy to answer questions about the architecture or MCP pooling implementation.
