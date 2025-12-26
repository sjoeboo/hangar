# Gemini CLI Integration Guide

## Overview

Agent-deck provides first-class support for Google's Gemini CLI with feature parity to Claude Code integration.

## How It Works

### Session Detection

Gemini sessions are stored in:
```
~/.gemini/tmp/<sha256-hash-of-project-path>/chats/session-*.json
```

Agent-deck:
1. Hashes your project path (SHA256 of absolute path)
2. Scans the chats directory
3. Detects sessions created after the instance started
4. Stores session ID for resume

### Session Capture

When you create a new Gemini session, agent-deck uses a capture-resume pattern:
```bash
session_id=$(gemini --output-format stream-json -i 2>/dev/null | head -1 | jq -r '.session_id') && \
tmux set-environment GEMINI_SESSION_ID "$session_id" && \
gemini --resume "$session_id"
```

This ensures the session ID is captured and stored in tmux environment for later use.

### Session Resume

Once a session ID is known, restart simply uses:
```bash
gemini --resume <session-id>
```

Press `r` or `R` in agent-deck to restart a Gemini session. This continues your conversation from where it left off.

### MCP Management

Press `M` on any Gemini session to manage MCP servers. Changes are written to `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "exa": {
      "command": "npx",
      "args": ["-y", "exa-mcp-server"],
      "env": {"EXA_API_KEY": "$YOUR_KEY"}
    }
  }
}
```

**Note:** Gemini only supports global MCPs (no project-level configuration).

### Response Extraction

```bash
agent-deck session output <session-id>
```

Extracts the last assistant response from the Gemini session JSON file. Agent-deck reads the session file directly and parses the message with `"type": "gemini"`.

## Limitations

| Feature | Supported? | Workaround |
|---------|-----------|------------|
| Session resume | Yes | `gemini --resume <id>` |
| Fork conversations | No | Use sub-sessions (create new session, link parent) |
| Project-level MCPs | No | All MCPs are global |
| Config dir override | No | Always uses `~/.gemini/` |

## Verified Technical Details

All implementation details verified against Gemini CLI:
- Project hash: SHA256 of absolute path (verified with `echo -n "/path" | shasum -a 256`)
- Session format: JSON with camelCase fields (`sessionId`, not `session_id`)
- Message type: `"type": "gemini"` (not `role: "assistant"`)
- Resume flag: `--resume <uuid>` or `--resume <index>`
- Session ID capture: `--output-format stream-json` provides immediate ID in first message

### Session File Format

```json
{
  "sessionId": "abc123-def456-...",
  "startTime": "2025-12-26T15:09:00.000Z",
  "lastUpdated": "2025-12-26T15:30:00.000Z",
  "messages": [
    {
      "id": "msg-1",
      "timestamp": "2025-12-26T15:09:01.000Z",
      "type": "user",
      "content": "Hello"
    },
    {
      "id": "msg-2",
      "timestamp": "2025-12-26T15:09:05.000Z",
      "type": "gemini",
      "content": "Hi! How can I help?",
      "model": "gemini-2.0-flash-exp",
      "toolCalls": [],
      "thoughts": []
    }
  ]
}
```

### Settings File Location

- **Global MCPs:** `~/.gemini/settings.json`
- **Session storage:** `~/.gemini/tmp/<project-hash>/chats/`

## Comparison with Claude

| Feature | Claude Code | Gemini CLI |
|---------|-------------|------------|
| Session Detection | tmux env + file scanning | tmux env + file scanning |
| Resume | `--resume <id>` | `--resume <id>` |
| Fork | `--fork-session <id>` | Not supported |
| MCP Scopes | Global + Project + Local | Global only |
| MCP Config File | `.claude.json` / `.mcp.json` | `settings.json` |
| Session Format | JSONL | JSON |
| Message Type Field | `role: "assistant"` | `type: "gemini"` |
| Config Dir Override | `CLAUDE_CONFIG_DIR` | Not supported |

## Usage Examples

### Create a New Gemini Session

1. Press `n` in agent-deck TUI
2. Select "gemini" as the tool
3. Enter a title and project path
4. Session starts with automatic ID capture

### Restart a Gemini Session

1. Select the Gemini session in the TUI
2. Press `r` or `R`
3. Session resumes with `gemini --resume <id>`

### Configure MCPs for Gemini

1. Select a Gemini session
2. Press `M` to open MCP Manager
3. Toggle MCPs on/off (all are global scope)
4. Press Enter to apply
5. Restart the session (`r`) to load new MCPs

### Get Session Output via CLI

```bash
# Get last response from a Gemini session
agent-deck session output my-gemini-session

# Get output as JSON
agent-deck session output my-gemini-session --json
```

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/session/gemini.go` | Session detection, project hashing, session listing |
| `internal/session/gemini_mcp.go` | MCP configuration for Gemini |
| `internal/session/instance.go` | `buildGeminiCommand()`, response extraction |

## Troubleshooting

### Session ID Not Detected

If the session ID is not captured:
1. Ensure `jq` is installed (`brew install jq`)
2. Check tmux environment: `tmux show-environment GEMINI_SESSION_ID`
3. Wait 10 seconds for auto-detection from session files

### MCPs Not Loading

If MCPs don't appear in Gemini:
1. Verify `~/.gemini/settings.json` was updated
2. Restart the Gemini session (`r`)
3. Check MCP server is correctly defined in `~/.agent-deck/config.toml`

### No Response Found

If `session output` returns "no response found":
1. Ensure the Gemini session has received at least one response
2. Check the session file exists in `~/.gemini/tmp/<hash>/chats/`
3. Verify the session ID matches an existing file
