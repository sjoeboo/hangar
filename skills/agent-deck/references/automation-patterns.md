# Automation & Scripting Patterns

Lightweight guide for automating agent-deck operations.

## JSON Output

All commands support `-json` flag. **Flags must come BEFORE arguments:**

```bash
# Correct
agent-deck session show -json my-project

# Incorrect (flag ignored)
agent-deck session show my-project -json
```

## Common Patterns

### Monitor Waiting Sessions

```bash
WAITING=$(agent-deck status -json | jq '.waiting')
if [ "$WAITING" -gt 0 ]; then
  echo "$WAITING sessions waiting"
fi
```

### Auto-Start Sessions

```bash
for session in api frontend database; do
  STATUS=$(agent-deck session show -json "$session" | jq -r '.status')
  if [ "$STATUS" != "running" ]; then
    agent-deck session start "$session"
  fi
done
```

### Bulk MCP Attachment

```bash
for mcp in exa github playwright; do
  agent-deck mcp attach my-project "$mcp"
done
agent-deck session restart my-project
```

## Exit Codes

```bash
0   # Success
1   # Generic error
2   # Not found
```

## Using with jq

```bash
# Get all session titles
agent-deck list -json | jq -r '.[].title'

# Get Claude sessions only
agent-deck list -json | jq -r '.[] | select(.tool == "claude") | .title'

# Count sessions per group
agent-deck list -json | jq 'group_by(.group) | map({group: .[0].group, count: length})'
```

## tmux Status Line

```bash
# Add to ~/.tmux.conf
set -g status-right '#(agent-deck status -q) waiting | %H:%M'
```
