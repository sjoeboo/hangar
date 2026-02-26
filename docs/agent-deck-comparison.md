# Hangar vs Agent Deck

Hangar is an opinionated fork of [agent-deck](https://ghe.spotify.net/mnicholson/agent-deck), stripped down to a clean, fast workflow for Claude Code users who live in git repos and worktrees.

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

Git history is preserved for easy upstream cherry-picks.
