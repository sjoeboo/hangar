# Troubleshooting Guide

Common issues and solutions for hangar.

## Quick Fixes

| Issue | Solution |
|-------|----------|
| Session shows `✕` error | `hangar session start <name>` |
| MCPs not loading | `hangar session restart <name>` |
| CLI changes not in TUI | Press `Ctrl+R` to refresh |
| Flag not working | Put flags BEFORE arguments |
| Fork fails | Check session has valid Claude session ID |
| Status stuck | Wait 2 seconds or press `u` to mark unread |

## Common Issues

### Flags Ignored

**Problem:** Flags after positional arguments are silently ignored.

```bash
# WRONG - message not sent
hangar session start my-project -m "Hello"

# CORRECT
hangar session start -m "Hello" my-project
```

### MCP Not Available

1. Check if attached: `hangar mcp attached <session>`
2. Restart session: `hangar session restart <session>`
3. Verify in config: `hangar mcp list`

### Session ID Not Detected

Claude session ID needed for fork/resume. Check:

```bash
hangar session show <name> --json | jq '.claude_session_id'
```

If null, restart session and interact with Claude.

### High CPU Usage

**With many sessions:** Normal if batched updates. Check:
```bash
hangar status  # Should show ~0.5% CPU when idle
```

**With active session:** Normal (live preview updates).

### Log Files Too Large

Add to `~/.hangar/config.toml`:
```toml
[logs]
max_size_mb = 1
max_lines = 2000
```

### Global Search Not Working

Check config:
```toml
[global_search]
enabled = true
```

Also verify `~/.claude/projects/` exists and has content.

## Debugging

Enable debug logging:
```bash
HANGAR_DEBUG=1 hangar
```

Check session logs:
```bash
tail -100 ~/.hangar/logs/agentdeck_<session>_*.log
```

## Report a Bug

If something isn't working, please create a GitHub issue with all relevant context.

### Step 1: Gather Information

Run these commands and save output:

```bash
# Version info
hangar version

# Current status
hangar status --json

# Session details (if session-related)
hangar session show <session-name> --json

# Config (sanitized - removes secrets)
cat ~/.hangar/config.toml | grep -v "KEY\|TOKEN\|SECRET\|PASSWORD"

# Recent logs (if error occurred)
tail -100 ~/.hangar/logs/agentdeck_<session>_*.log 2>/dev/null

# System info
uname -a
echo "tmux: $(tmux -V 2>/dev/null || echo 'not installed')"
```

### Step 2: Describe the Issue

Prepare clear answers to:

1. **What did you try?** (exact command or TUI action)
2. **What happened?** (error message, unexpected behavior)
3. **What did you expect?** (correct behavior)
4. **Can you reproduce it?** (steps to trigger)

### Step 3: Create GitHub Issue

Go to: **https://github.com/sjoeboo/hangar/issues/new**

Use this template:

```markdown
## Description

[Brief description of the issue]

## Steps to Reproduce

1. [First step]
2. [Second step]
3. [What happened]

## Expected Behavior

[What should have happened]

## Environment

- hangar version: [output of `hangar version`]
- OS: [macOS/Linux/WSL]
- tmux version: [output of `tmux -V`]

## Debug Output

<details>
<summary>Status JSON</summary>

```json
[paste hangar status --json]
```

</details>

<details>
<summary>Config (sanitized)</summary>

```toml
[paste sanitized config]
```

</details>

<details>
<summary>Logs</summary>

```
[paste relevant log lines]
```

</details>
```

### Step 4: Follow Up

- Check for responses on your issue
- Test any suggested fixes
- Update issue with results
- Join [Discord](https://discord.gg/e4xSs6NBN8) for quick help and community support

## Recovery

### Session Metadata Lost

Data stored in SQLite:
```bash
~/.hangar/profiles/default/state.db
```

Recovery (if state.db is corrupted):
```bash
# If sessions.json.migrated still exists, delete state.db and restart.
# hangar will auto-migrate from the .migrated file.
rm ~/.hangar/profiles/default/state.db
mv ~/.hangar/profiles/default/sessions.json.migrated \
   ~/.hangar/profiles/default/sessions.json
# Restart hangar to trigger auto-migration into a fresh state.db
```

### tmux Sessions Lost

Session logs preserved:
```bash
tail -500 ~/.hangar/logs/agentdeck_<session>_*.log
```

### Profile Corrupted

Create fresh:
```bash
hangar profile create fresh
hangar profile default fresh
```

## Uninstalling

Remove hangar from your system:

```bash
hangar uninstall              # Interactive uninstall
hangar uninstall --dry-run    # Preview what would be removed
hangar uninstall --keep-data  # Remove binary only, keep sessions
```

Or use the standalone script:
```bash
curl -fsSL https://raw.githubusercontent.com/sjoeboo/hangar/main/uninstall.sh | bash
```

**What gets removed:**
- **Binary:** `~/.local/bin/hangar` or `/usr/local/bin/hangar`
- **Homebrew:** `hangar` package (if installed via brew)
- **tmux config:** The `# hangar configuration` block in `~/.tmux.conf`
- **Data directory:** `~/.hangar/` (sessions, logs, config)

Use `--keep-data` to preserve your sessions and configuration.

## Critical Warnings

**NEVER run these commands - they destroy ALL hangar sessions:**

```bash
# DO NOT RUN
tmux kill-server
tmux ls | grep agentdeck | xargs tmux kill-session
```

**Recovery impossible** - metadata backups exist but tmux sessions are gone.
