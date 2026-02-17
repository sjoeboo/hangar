# I built a terminal dashboard to manage all my Claude Code sessions - Agent Deck

Hey r/ClaudeCode! I wanted to share something I've been building.

**The problem:** I was running 10+ Claude Code sessions at once (different projects, exploratory branches, background research agents) and constantly losing track of which tmux window was which. I'd have Claude sessions stuck "thinking" in forgotten tabs, eating up context tokens. I'd forget which MCPs were attached where. And when I wanted to fork a conversation to try two approaches, I was manually copy-pasting session IDs like a caveman.

**So I built Agent Deck** - a TUI (terminal UI) dashboard for managing AI coding sessions. It's tmux-based, written in Go, fully open source (MIT).

**What it does:**

- **Session forking**: Hit `F` on any Claude session, instantly fork it with full context inherited. Try multiple approaches in parallel without losing your place
- **MCP socket pooling**: Share MCP processes across sessions via Unix sockets. Went from 3.2GB memory for 5 sessions to 450MB (85-90% reduction)
- **Smart status detection**: Knows when Claude is thinking vs waiting for you
- **MCP Manager**: Press `M` to toggle MCP servers on/off per session without editing JSON files
- **Global search**: Search across all your Claude conversations at once
- **Works with multiple agents**: Claude Code, Gemini CLI, OpenCode, Codex

It supports groups, notifications, importing existing tmux sessions, and a bunch of other stuff. Currently at 772 stars on GitHub.

**Quick install:**
```bash
brew tap asheshgoplani/agent-deck
brew install agent-deck
agent-deck  # starts the TUI
```

GitHub: https://github.com/asheshgoplani/agent-deck

I've been using it daily for a few months now and it's completely changed how I work with Claude. Curious to hear what features would be useful for your workflows, or if you run into any issues!
