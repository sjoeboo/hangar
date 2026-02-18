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
# CREATE SESSION 1: docs (Claude - default)
# ============================================

Type "n"
Sleep 1s
Type "docs"
Tab
Sleep 300ms
Type "~/demo/docs"
Tab
Sleep 500ms
Enter
Sleep 2.5s

# ============================================
# CREATE SESSION 2: api (Gemini - Right once)
# ============================================

Type "n"
Sleep 1s
Type "api"
Tab
Sleep 300ms
Type "~/demo/api"
Tab
Sleep 300ms
Right
Sleep 500ms
Enter
Sleep 2.5s

# ============================================
# CREATE SESSION 3: bugs (Codex - Right 3x)
# ============================================

Type "n"
Sleep 1s
Type "bugs"
Tab
Sleep 300ms
Type "~/demo/bugs"
Tab
Sleep 300ms
Right
Right
Right
Sleep 500ms
Enter
Sleep 2.5s

# ============================================
# WORK: Enter "docs" (Claude), ask question, leave
# ============================================

# Wait for UI to settle
Sleep 500ms

# Navigate up to docs: bugs -> api -> docs
Type "k"
Sleep 400ms
Type "k"
Sleep 600ms

# Enter docs session
Enter

# Wait for Claude to fully initialize
Sleep 6s

# Ask question
Type "What are best practices for REST API design?"
Sleep 400ms
Enter
Sleep 3s

# Detach
Ctrl+Q
Sleep 1.5s

# ============================================
# WORK: Enter "api" (Gemini), ask question, leave
# ============================================

# Move down to api
Type "j"
Sleep 600ms

# Enter session
Enter
Sleep 3s

# Ask question
Type "Compare PostgreSQL vs MongoDB for user data"
Sleep 400ms
Enter
Sleep 3s

# Detach
Ctrl+Q
Sleep 1.5s

# ============================================
# RETURN: Go back to "docs" to see response
# ============================================

# Navigate up to docs
Type "k"
Sleep 600ms

# Enter to see response
Enter
Sleep 4s

# Detach
Ctrl+Q
Sleep 1.5s

# Show final dashboard
Sleep 2s
