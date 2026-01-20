#!/bin/bash
# import.sh - Import a shared Claude session file
#
# Usage: import.sh <file-path> [options]
#
# Options:
#   --title <name>     Override session title (default: from file)
#   --project <path>   Import to specific project (default: current directory)
#   --no-start         Don't start the session after import
#
# Examples:
#   import.sh ~/Downloads/session-2024-01-20-feature.json
#   import.sh session.json --title "Continued Work"
#   import.sh session.json --project /path/to/project

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

# Parse arguments
INPUT_FILE=""
TITLE_OVERRIDE=""
PROJECT_PATH=""
NO_START=false

while [ $# -gt 0 ]; do
    case "$1" in
        --title)
            TITLE_OVERRIDE="$2"
            shift 2
            ;;
        --project)
            PROJECT_PATH="$2"
            shift 2
            ;;
        --no-start)
            NO_START=true
            shift
            ;;
        *)
            if [ -z "$INPUT_FILE" ]; then
                INPUT_FILE="$1"
            fi
            shift
            ;;
    esac
done

if [ -z "$INPUT_FILE" ]; then
    echo "Usage: import.sh <file-path> [--title name] [--project path]" >&2
    exit 1
fi

# Resolve file path
if [[ "$INPUT_FILE" != /* ]]; then
    INPUT_FILE="$(pwd)/$INPUT_FILE"
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: File not found: $INPUT_FILE" >&2
    exit 1
fi

echo "Importing session from: $INPUT_FILE"

# Parse export file
echo "Reading export file..."

VERSION=$(jq -r '.version // "unknown"' "$INPUT_FILE")
EXPORTED_BY=$(jq -r '.exported_by // "unknown"' "$INPUT_FILE")
EXPORTED_AT=$(jq -r '.exported_at // "unknown"' "$INPUT_FILE")
SESSION_ID=$(jq -r '.session.id // empty' "$INPUT_FILE")
SESSION_TITLE=$(jq -r '.session.title // "Imported Session"' "$INPUT_FILE")
ORIGINAL_PROJECT=$(jq -r '.session.original_project // empty' "$INPUT_FILE")
SUMMARY=$(jq -r '.context.summary // ""' "$INPUT_FILE")

if [ -z "$SESSION_ID" ]; then
    echo "Error: Invalid export file - no session ID found" >&2
    exit 1
fi

echo ""
echo "Session Details:"
echo "  ID: $SESSION_ID"
echo "  Title: $SESSION_TITLE"
echo "  Exported by: $EXPORTED_BY"
echo "  Exported at: $EXPORTED_AT"
echo "  Original project: $ORIGINAL_PROJECT"
if [ -n "$SUMMARY" ]; then
    echo "  Summary: ${SUMMARY:0:100}..."
fi

# Use title override if provided
if [ -n "$TITLE_OVERRIDE" ]; then
    SESSION_TITLE="$TITLE_OVERRIDE"
fi

# Use current directory if no project specified
if [ -z "$PROJECT_PATH" ]; then
    PROJECT_PATH=$(pwd)
fi

echo ""
echo "Import destination: $PROJECT_PATH"

# Detect current profile
PROFILE=$(agent-deck session current --json 2>/dev/null | grep -v '^20' | jq -r '.profile // "default"' || echo "default")
echo "Profile: $PROFILE"

# Create the Claude projects directory for this project
ENCODED_PATH=$(encode_path "$PROJECT_PATH")
DEST_DIR="$(get_claude_projects_dir)/$ENCODED_PATH"
mkdir -p "$DEST_DIR"

# Extract and write JSONL
DEST_FILE="$DEST_DIR/$SESSION_ID.jsonl"

echo ""
echo "Writing session file to: $DEST_FILE"

# Check if file already exists
if [ -f "$DEST_FILE" ]; then
    echo "Warning: Session file already exists. Creating backup..."
    cp "$DEST_FILE" "$DEST_FILE.backup.$(date +%s)"
fi

# Extract messages and write as JSONL
if ! jq -c '.messages[]' "$INPUT_FILE" > "$DEST_FILE"; then
    echo "Error: Failed to extract messages from export file" >&2
    exit 1
fi

WRITTEN_LINES=$(wc -l < "$DEST_FILE" | tr -d ' ')
echo "Written $WRITTEN_LINES records"

# Create agent-deck session
echo ""
echo "Creating agent-deck session..."

# Build the session title with "Imported:" prefix
IMPORT_TITLE="Imported: $SESSION_TITLE"

# Check if session with this title already exists
EXISTING=$(agent-deck -p "$PROFILE" list --json 2>/dev/null | jq -r ".[] | select(.title == \"$IMPORT_TITLE\") | .id" || echo "")

if [ -n "$EXISTING" ]; then
    echo "Session '$IMPORT_TITLE' already exists. Updating..."
    # Just update the session to point to the new session ID
    agent-deck -p "$PROFILE" session set "$IMPORT_TITLE" claude-session-id "$SESSION_ID"
else
    # Create new session
    agent-deck -p "$PROFILE" add -t "$IMPORT_TITLE" -c claude "$PROJECT_PATH"

    # Wait a moment for session to be created
    sleep 1

    # Set the Claude session ID so it resumes the imported session
    agent-deck -p "$PROFILE" session set "$IMPORT_TITLE" claude-session-id "$SESSION_ID"
fi

echo ""
echo "=========================================="
echo "Session imported successfully!"
echo "=========================================="
echo ""
echo "Session: $IMPORT_TITLE"
echo "Claude Session ID: $SESSION_ID"
echo ""

if [ "$NO_START" = "true" ]; then
    echo "Session created but not started (--no-start)"
    echo ""
    echo "To start: agent-deck session start \"$IMPORT_TITLE\""
    echo "Or: Open agent-deck TUI and press Enter on the session"
else
    echo "Starting session..."
    agent-deck -p "$PROFILE" session start "$IMPORT_TITLE"

    echo ""
    echo "Session is now running. It will resume from the imported conversation."
    echo ""
    echo "To attach: agent-deck session attach \"$IMPORT_TITLE\""
    echo "Or: Open agent-deck TUI and press Enter on the session"
fi
