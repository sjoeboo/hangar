# CLI Command Reference

Complete reference for all hangar CLI commands.

## Table of Contents

- [Global Options](#global-options)
- [Basic Commands](#basic-commands)
- [Web Command](#web-command)
- [Session Commands](#session-commands)
- [MCP Commands](#mcp-commands)
- [Skill Commands](#skill-commands)
- [Group Commands](#group-commands)
- [Profile Commands](#profile-commands)
- [Conductor Commands](#conductor-commands)

## Global Options

```bash
-p, --profile <name>    Use specific profile
--json                  JSON output
-q, --quiet             Minimal output
```

## Basic Commands

### add - Create session

```bash
hangar add [path] [options]
```

| Flag | Description |
|------|-------------|
| `-t, --title` | Session title |
| `-g, --group` | Group path |
| `-c, --cmd` | Tool/command (claude, gemini, opencode, codex, custom) |
| `--wrapper` | Wrapper command; use `{command}` placeholder |
| `--parent` | Parent session (creates child) |
| `--no-parent` | Disable automatic parent linking |
| `--mcp` | Attach MCP (repeatable) |

```bash
hangar add -t "My Project" -c claude .
hangar add -t "Child" --parent "Parent" -c claude /tmp/x
hangar add -g ard --parent "conductor-ard" -c claude .
hangar add -c "codex --dangerously-bypass-approvals-and-sandbox" .
hangar add -t "Research" -c claude --mcp exa --mcp firecrawl /tmp/r
```

Notes:
- Parent auto-link is enabled by default when `HANGAR_INSTANCE_ID` is present and neither `--parent` nor `--no-parent` is passed.
- `--parent` and `--no-parent` are mutually exclusive.
- Explicit `-g/--group` overrides inherited parent group.
- If `--cmd` contains extra args and no explicit `--wrapper` is provided, hangar auto-generates a wrapper to preserve those args.

### launch - Create + start (+ optional message)

```bash
hangar launch [path] [options]
```

Examples:

```bash
hangar launch . -c claude -m "Review this module"
hangar launch . -g ard -c claude -m "Review dataset"
hangar launch . -c "codex --dangerously-bypass-approvals-and-sandbox"
```

### list - List sessions

```bash
hangar list [--json] [--all]
hangar ls  # Alias
```

### remove - Remove session

```bash
hangar remove <id|title>
hangar rm  # Alias
```

### status - Status summary

```bash
hangar status [-v|-q|--json]
```

- Default: `2 waiting - 5 running - 3 idle`
- `-v`: Detailed list by status
- `-q`: Just waiting count (for scripts)

## Web Command

### web - Start browser UI

```bash
hangar web [options]
```

| Flag | Description |
|------|-------------|
| `--listen` | Listen address (default: `127.0.0.1:8420`) |
| `--read-only` | Disable terminal input, stream output only |
| `--token` | Require bearer token for API and WS access |
| `--open` | Reserved placeholder (currently no-op) |

```bash
hangar web
hangar web --read-only
hangar web --token my-secret
hangar -p work web --listen 127.0.0.1:9000
```

When token auth is enabled, open the web UI with:

```bash
http://127.0.0.1:8420/?token=my-secret
```

## Session Commands

### session start

```bash
hangar session start <id|title> [-m "message"] [--json] [-q]
```

`-m` sends initial message after agent is ready.
Flags can be placed before or after the session identifier.

### session stop

```bash
hangar session stop <id|title>
```

### session restart

```bash
hangar session restart <id|title>
```

Reloads MCPs without losing conversation (Claude/Gemini).

### session fork (Claude only)

```bash
hangar session fork <id|title> [-t "title"] [-g "group"]
```

Creates new session with same Claude conversation.

**Requirements:**
- Session must be Claude tool
- Must have valid Claude session ID

### session attach

```bash
hangar session attach <id|title>
```

Interactive PTY mode. Press `Ctrl+Q` to detach.

### session show

```bash
hangar session show [id|title] [--json] [-q]
```

Auto-detects current session if no ID provided.

**JSON output includes:**
- Session details (id, title, status, path, group, tool)
- Claude/Gemini session ID
- Attached MCPs (local, global, project)
- tmux session name

### session current

```bash
hangar session current [--json] [-q]
```

Auto-detect current session and profile from tmux environment.

```bash
# Human-readable
hangar session current
# Session: test, Profile: work, ID: c5bfd4b4, Status: running

# For scripts
hangar session current -q
# test

# JSON
hangar session current --json
# {"session":"test","profile":"work","id":"c5bfd4b4",...}
```

**Profile auto-detection priority:**
1. `HANGAR_PROFILE` env var
2. Parse from `CLAUDE_CONFIG_DIR` (`~/.claude-work` -> `work`)
3. Config default or `default`

### session set

```bash
hangar session set <id|title> <field> <value>
```

**Fields:** title, path, command, tool, claude-session-id, gemini-session-id

### session send

```bash
hangar session send <id|title> "message" [--no-wait] [-q] [--json]
```

Default behavior:
- Waits for agent readiness before sending.
- Verifies processing starts after send.
- If Claude leaves a pasted prompt unsent (`[Pasted text ...]`), retries `Enter` automatically.
- Avoids unnecessary retry `Enter` presses when session is already `waiting`/`idle`.

### session output

```bash
hangar session output [id|title] [--json] [-q]
```

Get last response from Claude/Gemini session.

### session set-parent / unset-parent

```bash
hangar session set-parent <session> <parent>
hangar session unset-parent <session>
```

## MCP Commands

### mcp list

```bash
hangar mcp list [--json] [-q]
```

### mcp attached

```bash
hangar mcp attached [id|title] [--json] [-q]
```

Shows MCPs from LOCAL, GLOBAL, PROJECT scopes.

### mcp attach

```bash
hangar mcp attach <session> <mcp> [--global] [--restart]
```

- `--global`: Write to Claude config (all projects)
- `--restart`: Restart session immediately

### mcp detach

```bash
hangar mcp detach <session> <mcp> [--global] [--restart]
```

## Skill Commands

Skills are discovered from configured sources and attached per project (Claude only).

### skill list

```bash
hangar skill list [--source <name>] [--json] [-q]
hangar skill ls
```

`--source` filters by source name (for example `pool`, `claude-global`, `team`).

### skill attached

```bash
hangar skill attached [id|title] [--json] [-q]
```

Shows:
- Manifest-managed attachments from `<project>/.hangar/skills.toml`
- Unmanaged entries currently present in `<project>/.claude/skills`

### skill attach

```bash
hangar skill attach <session> <skill> [--source <name>] [--restart] [--json] [-q]
```

- `--source`: Force source when name is ambiguous
- `--restart`: Restart session immediately after attach

### skill detach

```bash
hangar skill detach <session> <skill> [--source <name>] [--restart] [--json] [-q]
```

- `--source`: Filter by source when detaching
- `--restart`: Restart session immediately after detach

### skill source list

```bash
hangar skill source list [--json] [-q]
hangar skill source ls
```

### skill source add

```bash
hangar skill source add <name> <path> [--description "..."] [--json] [-q]
```

### skill source remove

```bash
hangar skill source remove <name> [--json] [-q]
hangar skill source rm <name>
```

## Group Commands

### group list

```bash
hangar group list [--json] [-q]
```

### group create

```bash
hangar group create <name> [--parent <group>]
```

### group delete

```bash
hangar group delete <name> [--force]
```

`--force`: Move sessions to parent and delete.

### group move

```bash
hangar group move <session> <group>
```

Use `""` or `root` to move to default group.

## Profile Commands

```bash
hangar profile list
hangar profile create <name>
hangar profile delete <name>
hangar profile default [name]
```

## Conductor Commands

```bash
hangar conductor setup <name> [--description "..."] [--heartbeat|--no-heartbeat]
hangar conductor teardown <name> [--remove]
hangar conductor teardown --all [--remove]
hangar conductor status [name]
hangar conductor list [--profile <name>]
```

- `setup` creates `~/.hangar/conductor/<name>/` plus `meta.json` and registers `conductor-<name>` session in the selected profile.
- `setup` also installs shared `~/.hangar/conductor/CLAUDE.md` (or symlink via `--shared-claude-md`).
- Heartbeat timers run per conductor (default every 15 minutes) and can be disabled with `--no-heartbeat`.
- Heartbeat sends use non-blocking `session send --no-wait -q` to avoid timeout churn when sessions are busy.
- Bridge daemon is installed only when Telegram and/or Slack is configured in `[conductor]`.
- Transition notifier daemon (`hangar notify-daemon`) is installed by setup and sends event nudges on `running -> waiting|error|idle` transitions (parent first, then conductor fallback).

## Session Resolution

Commands accept:
- **Title:** `"My Project"` (exact match)
- **ID prefix:** `abc123` (6+ chars)
- **Path:** `/path/to/project`
- **Current:** Omit ID in tmux (uses env var)

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error |
| 2 | Not found |
