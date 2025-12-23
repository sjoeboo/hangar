# Agent Deck - Overview Demo
# Shows: Create sessions, work in parallel, switch between them

Output agent-deck-overview.gif
Output agent-deck-overview.mp4
Output agent-deck-overview.webm

Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 18
Set Width 1400
Set Height 800
Set TypingSpeed 75ms
Set Padding 20
Set Margin 20
Set MarginFill "#1a1b26"
Set BorderRadius 10
Set WindowBar Colorful
Set WindowBarSize 40
Set Framerate 30
Set PlaybackSpeed 1.4

# Hidden: Clean start
Hide
Type "rm -rf ~/.agent-deck/profiles/demo && mkdir -p ~/demo/{docs,api,bugs}"
Enter
Sleep 500ms
Show

# ============================================
# DEMO START: Launch agent-deck
# ============================================

Sleep 500ms
Type "agent-deck -p demo"
Enter
Sleep 3s

# Show empty state briefly
Sleep 1.5s

# ============================================
# CREATE SESSION 1: docs (Claude - default, no Right needed)
# ============================================

Type "n"
Sleep 1s

# Name
Type "docs"
Sleep 300ms

# Tab to path
Tab
Sleep 300ms
Type "~/demo/docs"
Sleep 300ms

# Tab to command - Claude is default, just Enter
Tab
Sleep 500ms

# Create
Enter
Sleep 2.5s

# ============================================
# CREATE SESSION 2: api (Gemini - Right once)
# ============================================

Type "n"
Sleep 1s

# Name
Type "api"
Sleep 300ms

# Tab to path
Tab
Sleep 300ms
Type "~/demo/api"
Sleep 300ms

# Tab to command, Right once for Gemini
Tab
Sleep 300ms
Right
Sleep 500ms

# Create
Enter
Sleep 2.5s

# ============================================
# CREATE SESSION 3: bugs (Codex - Right three times)
# ============================================

Type "n"
Sleep 1s

# Name
Type "bugs"
Sleep 300ms

# Tab to path
Tab
Sleep 300ms
Type "~/demo/bugs"
Sleep 300ms

# Tab to command, Right 3x for Codex
Tab
Sleep 300ms
Right
Sleep 200ms
Right
Sleep 200ms
Right
Sleep 500ms

# Create
Enter
Sleep 2.5s

# ============================================
# WORK: Enter "docs" (Claude), ask question, leave
# ============================================

# Navigate to docs (first session)
Up
Sleep 300ms
Up
Sleep 500ms

# Enter session
Enter

# Wait for Claude to initialize and show prompt
Sleep 5s

# Ask question
Type "What are best practices for REST API design?"
Sleep 400ms
Enter
Sleep 2.5s

# Detach
Ctrl+Q
Sleep 1.5s

# ============================================
# WORK: Enter "api" (Gemini), ask question, leave
# ============================================

# Move down to api
Down
Sleep 500ms

# Enter session
Enter
Sleep 3s

# Ask question
Type "Compare PostgreSQL vs MongoDB for user data"
Sleep 400ms
Enter
Sleep 2.5s

# Detach
Ctrl+Q
Sleep 1.5s

# ============================================
# RETURN: Go back to "docs" to see Claude's response
# ============================================

# Navigate up to docs
Up
Sleep 500ms

# Enter to see response
Enter
Sleep 4s

# Detach
Ctrl+Q
Sleep 1.5s

# Show final dashboard
Sleep 2s
