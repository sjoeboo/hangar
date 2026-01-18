# Agent Deck

Terminal session manager for AI coding agents. Built with Go + Bubble Tea. **Version**: 0.8.14

## Agent Deck Skill (for Claude Code)

```bash
# Install: /plugin marketplace add asheshgoplani/agent-deck && /plugin install agent-deck@agent-deck
# Update:  /plugin marketplace update agent-deck
```

**Skill location:** `skills/agent-deck/` | **Update:** Edit files → commit → push → users run update command

---

## GitHub Issue Auto-Analysis

When a `.github-issue-context.json` file exists in the project root, you are in an **automated issue analysis session**. The file contains GitHub issue context sent via the notification system.

**Your task:**
1. Read and understand the issue from the context file
2. Search the codebase for related files using Glob/Grep
3. Analyze what changes would be needed
4. Create a clear action plan
5. DO NOT make any changes - only analyze and plan

**Context file structure:**
```json
{
  "number": 123,
  "title": "Issue title",
  "body": "Issue description...",
  "labels": ["bug", "enhancement"],
  "author": "username",
  "url": "https://github.com/...",
  "detected_type": "bug|feature|docs|support",
  "related_issues": [...],
  "recent_commits": [...]
}
```

**After analysis, clean up:** `rm .github-issue-context.json`

---

## GitHub PR Auto-Analysis

When a `.github-pr-context.json` file exists in the project root, you are in an **automated PR review session**. The file contains full PR context including diffs, commits, and review comments.

**Your task:**
1. Read the PR context file thoroughly
2. Understand what the PR is trying to accomplish
3. Review each changed file against the codebase
4. Check for:
   - Code correctness and logic errors
   - Consistency with existing patterns
   - Potential bugs or edge cases
   - Missing tests or documentation
   - Security concerns
5. Provide a detailed review with specific feedback
6. DO NOT approve or merge - only analyze and report

**Context file structure:**
```json
{
  "type": "pull_request",
  "number": 42,
  "title": "PR title",
  "body": "PR description...",
  "author": "username",
  "base_branch": "main",
  "head_branch": "feature/xyz",
  "additions": 150,
  "deletions": 30,
  "files": [
    {
      "filename": "path/to/file.go",
      "status": "modified",
      "additions": 50,
      "deletions": 10,
      "patch": "diff content..."
    }
  ],
  "commits": [{"sha": "abc123", "message": "..."}],
  "reviews": [],
  "review_comments": [],
  "detected_type": "feature|bugfix|docs|refactor"
}
```

**Review checklist:**
- [ ] Changes match PR description
- [ ] Code follows project conventions
- [ ] No obvious bugs or logic errors
- [ ] Edge cases handled
- [ ] Tests included (if applicable)
- [ ] Documentation updated (if applicable)

**After analysis, clean up:** `rm .github-pr-context.json`

---

## CRITICAL: Data Protection Rules

**THIS SECTION MUST NEVER BE DELETED OR IGNORED**

### tmux Session Loss Prevention

**NEVER DO THESE:**
1. `tmux kill-server` - Destroys ALL sessions instantly
2. `tmux kill-session` with patterns (e.g., `tmux ls | grep agentdeck | xargs ...`) - DESTROYS ALL
3. Quit Terminal completely while sessions running
4. Restart macOS without exporting outputs
5. Run tests that interfere with production tmux
6. Run cleanup commands targeting "agentdeck" patterns

**Incidents:** 2025-12-09 (37 sessions), 2025-12-10 (40 sessions killed by Claude in dangerous mode)

**Recovery:** Logs at `~/.agent-deck/logs/`, backups at `~/.agent-deck/profiles/default/sessions.json.bak{,.1,.2}`

### config.toml Protection (26 MCPs with API Keys)

**NEVER:** Delete, overwrite, "simplify", or reset `~/.agent-deck/config.toml` - Contains ALL MCP definitions with API keys (EXA, YouTube, Notion, Firecrawl, Twitter OAuth, Neo4j, GitHub, etc.)

**BEFORE ANY EDIT:** `cp ~/.agent-deck/config.toml ~/.agent-deck/config.toml.backup-$(date +%Y%m%d-%H%M%S)`

**Incident (2026-01-18):** config.toml reduced to minimal, lost all 26 MCPs. Recovery required searching conversation history.

### Test Isolation

Tests MUST use `_test` profile. **NEVER DELETE these TestMain files:**
- `internal/ui/testmain_test.go`
- `internal/tmux/testmain_test.go`
- `cmd/agent-deck/testmain_test.go`
- `internal/session/testmain_test.go`

### GitHub Actions Require Permission

**NEVER** post to GitHub (issues, PRs, comments) without explicit user permission. Always draft, show, wait for "yes, post it".

### Public Repository - NO Private Data

**ALL personal docs → `docs/` folder** (excluded via `.git/info/exclude`)

**NEVER push:** API keys, tokens, `~/.agent-deck/config.toml`, session logs, personal markdown files

**See:** [docs/BUG_FIXES.md](docs/BUG_FIXES.md) for detailed incident history and fixes.

---

## Quick Start

```bash
make build      # Build to ./build/agent-deck
make test       # Run tests
```

**Dependencies:** `brew install tmux jq`
**Dev symlink:** `sudo ln -sf /Users/ashesh/claude-deck/build/agent-deck /usr/local/bin/agent-deck`

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
# Or: brew install asheshgoplani/tap/agent-deck
# Or: go install github.com/asheshgoplani/agent-deck/cmd/agent-deck@latest
```

---

## CLI Commands

```bash
agent-deck                    # Interactive TUI
agent-deck add /path -t "Title" -g "group" -c claude
agent-deck list [--json]      # List sessions
agent-deck remove <id|title>  # Remove session
agent-deck status [-v|-q|--json]
agent-deck version / help
```

### Session Commands

| Command | Description |
|---------|-------------|
| `session start/stop/restart <id>` | Control session (restart resumes Claude/Gemini with updated MCPs) |
| `session fork <id>` | Fork Claude session (`-t/--title`, `-g/--group`) |
| `session attach <id>` | Attach to session (PTY mode) |
| `session show [id]` | Show details (current if no id) |
| `session current` | Auto-detect current session/profile (`-q`, `--json`) |
| `session set <id> <field> <value>` | Update property (title, path, command, tool, claude-session-id) |
| `session send <id> <msg>` | Send message to running session |
| `session output [id]` | Get last response |

### MCP Commands

| Command | Description |
|---------|-------------|
| `mcp list` | List available MCPs |
| `mcp attached [id]` | List MCPs attached to session |
| `mcp attach/detach <id> <mcp>` | Attach/detach MCP (`--global`, `--restart` for Claude) |

### Group Commands

| Command | Description |
|---------|-------------|
| `group list/create/delete/move` | Manage groups (`--parent`, `--force` flags) |

### Try Commands (Quick Experiments)

| Command | Description |
|---------|-------------|
| `try <name>` | Find/create experiment, start session |
| `try --list [query]` | List/search experiments |

### Global Flags

`--json` (automation) | `-q/--quiet` (minimal) | `-p/--profile` (profile)

### Session Resolution

Commands accept: **Title** (exact/fuzzy), **ID prefix**, **Path**, or omit ID in tmux for current session.

---

## Project Structure

```
cmd/agent-deck/     # CLI: main.go, session_cmd.go, mcp_cmd.go, group_cmd.go, cli_utils.go
internal/
├── ui/             # TUI: home.go, styles.go, dialogs, help, search, menu, preview, tree
├── platform/       # Platform detection (WSL1/WSL2/macOS/Linux)
├── profile/        # Profile auto-detection
├── session/        # Data: instance.go, groups.go, storage.go, config.go, userconfig.go
└── tmux/           # tmux: tmux.go, detector.go, pty.go
```

---

## Keyboard Shortcuts

### Navigation
`j/↓` `k/↑` Move | `h/←` Collapse | `l/→/Tab` Expand/collapse

### Session Actions
| Key | Action |
|-----|--------|
| `Enter` | Attach/toggle group |
| `n` | New session |
| `r/R` | Restart (resumes with updated MCPs) |
| `M` | MCP Manager |
| `e` | Rename |
| `m` | Move to group |
| `d` | Delete |
| `u` | Mark unread |
| `K/J` | Reorder |
| `f/F` | Fork (Claude only, quick/dialog) |

### View Actions
| Key | Action |
|-----|--------|
| `g` | New group |
| `/` | Search |
| `G` | Global Search |
| `?` | Help |
| `i` | Import tmux sessions |

### Quick Filters
`0` All | `!` Running | `@` Waiting | `#` Idle | `$` Error

### Global
`Ctrl+Q` Detach | `q/Ctrl+C` Quit

---

## MCP Manager

Press `M` to attach/detach MCP servers.

**Claude:** LOCAL (`.mcp.json`) or GLOBAL (Claude config)
**Gemini:** Global only (`~/.gemini/settings.json`)

**Controls:** `Tab` scope | `←/→` columns | `Space` toggle | `Enter` apply | `Esc` cancel

**Indicators:** `(l)` LOCAL | `(g)` GLOBAL | `(p)` PROJECT | `⟳` pending restart | `✕` stale

---

## Core Concepts

### Session Status

| Status | Symbol | Color | Meaning |
|--------|--------|-------|---------|
| Running | `●` | Green #9ece6a | Busy or content changed <2s |
| Waiting | `◐` | Yellow #e0af68 | Stopped, unacknowledged |
| Idle | `○` | Gray #565f89 | Stopped, acknowledged |
| Error | `✕` | Red #f7768e | Session doesn't exist |

### Storage

`~/.agent-deck/sessions.json` contains instances and groups. Groups use path-based hierarchy: `"parent/child/grandchild"`.

---

## Configuration (`~/.agent-deck/config.toml`)

```toml
[claude]
config_dir = "~/.claude-work"  # Custom profile
dangerous_mode = true          # --dangerously-skip-permissions

[logs]
max_size_mb = 1
max_lines = 2000
remove_orphans = true

[global_search]
enabled = true
recent_days = 30
tier = "auto"

[notifications]
enabled = true
max_shown = 6

[mcps.example]
command = "npx"
args = ["-y", "package-name"]
env = { API_KEY = "..." }
description = "Description"

[mcp_pool]
enabled = true
pool_all = true
exclude_mcps = []
fallback_to_stdio = true

# Custom CLI tools - see docs/INTERNALS.md for full reference
[tools.my-ai]
command = "my-ai"
icon = "🧠"
busy_patterns = ["thinking..."]
prompt_patterns = ["> "]
dangerous_flag = "--yes"
```

---

## Gemini CLI Integration

- **Session Detection:** Automatic from `~/.gemini/tmp/<hash>/chats/`
- **Resume:** Press `r` → `gemini --resume <id>`
- **MCP:** Press `M` (global only)
- **Limitations:** No fork, global MCPs only

See `GEMINI_INTEGRATION.md` for comprehensive guide.

---

## Fork Session (Claude Only)

Press `f` (quick) or `F` (dialog). Requires valid `lastSessionId` in Claude config.

**Subagent --add-dir:** With `--parent`, subagents get `--add-dir /path/to/parent/project` automatically.

---

## Global Search

Press `G` to search ALL Claude conversations in `~/.claude/projects/`.

**Controls:** `↑/↓` navigate | `[/]` scroll preview | `Enter` open | `Tab` local | `Esc` close

---

## Notification Bar

Shows waiting sessions in tmux status bar: `⚡ [1] frontend [2] api`

Press `Ctrl+b 1-6` to quick-switch. Config: `[notifications] enabled = true`

---

## Colors (Tokyo Night)

| Color | Hex | Use |
|-------|-----|-----|
| Accent | #7aa2f7 | Selection |
| Green | #9ece6a | Running |
| Yellow | #e0af68 | Waiting |
| Red | #f7768e | Error |
| Cyan | #7dcfff | Groups |
| Purple | #bb9af7 | Tool tags |

---

## Development

### Add Keyboard Shortcut
1. `home.go` → `Update()`, add `case "key":`
2. Update `renderHelpBar()`

### Add Dialog
1. Create `internal/ui/mydialog.go` with Show/Hide/IsVisible/Update/View
2. Add to Home struct, init in NewHome()
3. Check IsVisible() in Home.Update() and Home.View()

### Testing
```bash
go test ./...                     # All
go test ./internal/session/... -v # Session
go test ./internal/ui/... -v      # UI
```

### Debugging
```bash
AGENTDECK_DEBUG=1 agent-deck  # Logs status transitions
```

---

## Release

```bash
# 1. Update version in cmd/agent-deck/main.go
git commit -m "chore: bump version to vX.Y.Z" && git push origin main
git tag vX.Y.Z && git push origin vX.Y.Z  # Triggers release
```

**CI:** `.github/workflows/ci.yml` | **Release:** `.github/workflows/release.yml` (GoReleaser)

---

## Documentation

### Detailed References (in `docs/` folder, git-ignored)

| File | Contents |
|------|----------|
| [docs/BUG_FIXES.md](docs/BUG_FIXES.md) | Detailed bug history, root causes, fixes, incidents |
| [docs/INTERNALS.md](docs/INTERNALS.md) | Status detection, MCP pool, custom CLI tools, performance |

### Documentation Update Policy

**IMPORTANT:** When making changes to agent-deck, keep documentation in sync:

1. **This file (CLAUDE.md)** - Quick reference, CLI commands, keyboard shortcuts
2. **docs/BUG_FIXES.md** - Add new bug fixes with symptom/cause/fix
3. **docs/INTERNALS.md** - Update technical details when internals change
4. **README.md** - Public-facing docs
5. **skills/agent-deck/SKILL.md** - Update skill workflows

**Checklist for new features:**
- [ ] Update CLAUDE.md command tables
- [ ] Update README.md
- [ ] Update skill if affects CLI usage
- [ ] Add bug fixes to docs/BUG_FIXES.md if applicable
