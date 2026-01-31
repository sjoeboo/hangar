#!/bin/bash
# launch-subagent.sh - Launch a sub-agent as child of current session
#
# Usage: launch-subagent.sh "Title" "Prompt" [options]
#
# Options:
#   --mcp <name>     Attach MCP (can repeat)
#   --tool <type>    Agent tool: claude, codex, gemini (default: claude)
#   --wait           Poll until complete, return output
#   --timeout <sec>  Wait timeout (default: 300)
#
# Examples:
#   launch-subagent.sh "Research" "Find info about X"
#   launch-subagent.sh "Task" "Do Y" --mcp exa --mcp firecrawl
#   launch-subagent.sh "Query" "Answer Z" --wait --timeout 120
#   launch-subagent.sh "Consult" "Review this approach" --tool codex --wait

set -e

# Parse arguments
TITLE=""
PROMPT=""
TOOL="claude"
MCPS=()
WAIT=false
TIMEOUT=300

while [ $# -gt 0 ]; do
    case "$1" in
        --mcp)
            MCPS+=("$2")
            shift 2
            ;;
        --tool)
            TOOL="$2"
            shift 2
            ;;
        --wait)
            WAIT=true
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            if [ -z "$TITLE" ]; then
                TITLE="$1"
            elif [ -z "$PROMPT" ]; then
                PROMPT="$1"
            fi
            shift
            ;;
    esac
done

if [ -z "$TITLE" ] || [ -z "$PROMPT" ]; then
    echo "Usage: launch-subagent.sh \"Title\" \"Prompt\" [--tool codex] [--mcp name] [--wait]" >&2
    exit 1
fi

# Detect current session (filter out log lines starting with year)
CURRENT_JSON=$(agent-deck session current --json 2>/dev/null | grep -v '^20')
PARENT=$(echo "$CURRENT_JSON" | jq -r '.session')
PROFILE=$(echo "$CURRENT_JSON" | jq -r '.profile')

if [ -z "$PARENT" ] || [ "$PARENT" = "null" ]; then
    echo "Error: Not in an agent-deck session" >&2
    exit 1
fi

# Create work directory
SAFE_TITLE=$(echo "$TITLE" | tr ' ' '-' | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]-')
WORK_DIR="/tmp/${SAFE_TITLE}"
mkdir -p "$WORK_DIR"

# Build add command
ADD_CMD="agent-deck -p $PROFILE add -t \"$TITLE\" --parent \"$PARENT\" -c $TOOL"
for mcp in "${MCPS[@]}"; do
    ADD_CMD="$ADD_CMD --mcp $mcp"
done
ADD_CMD="$ADD_CMD \"$WORK_DIR\""

# Create and start session
eval "$ADD_CMD"
agent-deck -p "$PROFILE" session start "$TITLE"

# Get tmux session name for readiness check
TMUX_SESSION=$(agent-deck -p "$PROFILE" session show "$TITLE" 2>/dev/null | grep '^Tmux:' | awk '{print $2}')

# Tool-aware readiness patterns
case "$TOOL" in
    codex)
        READY_PATTERN="OpenAI Codex|>_|codex>|How can I help"
        ;;
    gemini)
        READY_PATTERN="Gemini|gemini|>>>"
        ;;
    *)
        READY_PATTERN=">|claude|Claude Code|/tmp/"
        ;;
esac

# Wait for agent to be ready
echo "Waiting for $TOOL to initialize..."
for i in $(seq 1 20); do
    PANE_CONTENT=$(tmux capture-pane -t "$TMUX_SESSION" -p 2>/dev/null | tail -5)

    if echo "$PANE_CONTENT" | grep -qE "($READY_PATTERN)" 2>/dev/null; then
        sleep 2  # Extra buffer for stability
        break
    fi
    sleep 1
done

# Send prompt
agent-deck -p "$PROFILE" session send "$TITLE" "$PROMPT"

# Codex safety net: Ink TUI sometimes needs an extra Enter nudge
if [ "$TOOL" = "codex" ]; then
    sleep 1
    tmux send-keys -t "$TMUX_SESSION" Enter
fi

echo ""
echo "Sub-agent launched:"
echo "  Title:   $TITLE"
echo "  Tool:    $TOOL"
echo "  Parent:  $PARENT"
echo "  Profile: $PROFILE"
echo "  Path:    $WORK_DIR"
if [ ${#MCPS[@]} -gt 0 ]; then
    echo "  MCPs:    ${MCPS[*]}"
fi
echo ""
echo "Check output with: agent-deck session output \"$TITLE\""

# If --wait, poll until complete
if [ "$WAIT" = "true" ]; then
    echo ""
    echo "Waiting for completion (timeout: ${TIMEOUT}s)..."

    START_TIME=$(date +%s)
    while true; do
        STATUS=$(agent-deck -p "$PROFILE" session show "$TITLE" 2>/dev/null | grep '^Status:' | awk '{print $2}')

        if [ "$STATUS" = "◐" ] || [ "$STATUS" = "waiting" ]; then
            echo "Complete!"
            echo ""
            echo "=== Response ==="
            agent-deck -p "$PROFILE" session output "$TITLE"
            exit 0
        fi

        ELAPSED=$(($(date +%s) - START_TIME))
        if [ $ELAPSED -ge $TIMEOUT ]; then
            echo "Timeout after ${TIMEOUT}s (session still running)" >&2
            echo "Check later with: agent-deck session output \"$TITLE\""
            exit 1
        fi

        sleep 5
    done
fi
