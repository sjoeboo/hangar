# Agent Deck CLI Command Reference

Complete reference for all agent-deck CLI commands, flags, and options.

## Table of Contents

- [Global Options](#global-options)
- [Basic Commands](#basic-commands)
- [Session Commands](#session-commands)
- [MCP Commands](#mcp-commands)
- [Group Commands](#group-commands)
- [Profile Commands](#profile-commands)
- [Output Modes](#output-modes)
- [Session Resolution](#session-resolution)

## Global Options

These flags work with any command:

```bash
-p <profile>, --profile=<profile>    # Use specific profile
```

**Examples:**
```bash
agent-deck -p work list
agent-deck --profile=production session start my-app
```

## Basic Commands

### add - Add a new session

```bash
agent-deck add [path] [options]
```

**Arguments:**
- `[path]` - Project directory (defaults to current directory if omitted)

**Options:**
- `-t <title>`, `--title=<title>` - Session title (defaults to folder name)
- `-g <group>`, `--group=<group>` - Group path (defaults to parent folder)
- `-c <command>`, `--cmd=<command>` - Command to run (claude, gemini, aider, codex, cursor, custom)

**Examples:**
```bash
agent-deck add                                # Use current directory
agent-deck add /path/to/project               # Add specific path
agent-deck add -t "My Project" -g work        # With title and group (current dir)
agent-deck add -t "My Project" -g work .      # Explicit current directory
agent-deck add -c claude                      # Specify command
agent-deck -p work add                        # Add to 'work' profile
```

### list - List all sessions

```bash
agent-deck list [options]
agent-deck ls    # Alias
```

**Options:**
- `-json` - Output as JSON
- `-all` - List sessions from all profiles

### status - Show session status summary

```bash
agent-deck status [options]
```

**Options:**
- `-v`, `-verbose` - Show detailed session list
- `-q`, `-quiet` - Only output waiting count (for scripts)
- `-json` - Output as JSON

## Session Commands

All session commands support `-json` and `-q`/`-quiet` flags.

### session start

```bash
agent-deck session start [options] <id|title>
```

**Options:**
- `-m <message>`, `--message=<message>` - Initial message to send once agent is ready

**CRITICAL:** All flags must come BEFORE the session name!

### session stop / restart / fork / attach / show / current

See main SKILL.md for usage patterns.

## MCP Commands

### mcp list / attached / attach / detach

See [mcp-management.md](mcp-management.md) for complete guide.

## Group Commands

### group list / create / delete / move

See main SKILL.md for usage patterns.

## Profile Commands

### profile list / create / delete / default

See [profiles.md](profiles.md) for complete guide.

## Output Modes

**CRITICAL:** Flags MUST come BEFORE positional arguments!

```bash
# Correct
agent-deck session show -json my-project

# WRONG (flag ignored!)
agent-deck session show my-project -json
```

## Session Resolution

Commands accept:
- **Title:** `"My Project"`
- **ID prefix:** `abc123` (>=6 chars)
- **Path:** `/Users/me/project`
- **Auto-detect:** Omit ID when inside tmux session

## Environment Variables

```bash
AGENTDECK_PROFILE=work    # Default profile
AGENTDECK_DEBUG=1         # Enable debug logging
```

## Exit Codes

```bash
0   # Success
1   # Generic error
2   # Not found
```
