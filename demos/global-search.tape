# Agent Deck - Global Search Demo
# Story: "Never Lose a Conversation" - Delete a session, recover it with Global Search

Output global-search.gif
Output global-search.mp4
Output global-search.webm

# Terminal settings
Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 18
Set Width 1400
Set Height 900
Set TypingSpeed 40ms
Set Padding 20
Set Margin 20
Set MarginFill "#1a1b26"
Set BorderRadius 10
Set WindowBar Colorful
Set WindowBarSize 40
Set Framerate 30
Set PlaybackSpeed 1.3

# ============================================
# SETUP: Create a session with conversation (hidden)
# IMPORTANT: Long waits to ensure Claude responds and saves
# ============================================
Hide

# Clean demo profile
Type "rm -rf ~/.agent-deck/profiles/search-demo && mkdir -p ~/demo-project"
Enter
Sleep 500ms

# Start agent-deck, create a session
Type "agent-deck -p search-demo"
Enter
Sleep 2s

# Create new session
Type "n"
Sleep 1s
Type "API Authentication"
Tab
Type "~/demo-project"
Tab
Enter
Sleep 12s

# Enter the session and wait for Claude to fully start
Type "j"
Sleep 500ms
Enter
Sleep 20s

# Start a conversation about JWT
Type "Explain JWT token validation best practices"
Sleep 300ms
Enter

# CRITICAL: Wait long enough for Claude to FULLY respond and save
# Claude needs time to: process, respond, and write to .jsonl file
Sleep 90s

# Detach
Ctrl+Q
Sleep 1s

# Quit agent-deck
Type "q"
Sleep 500ms

Show

# ============================================
# ACT 1: Show dashboard with the session
# ============================================
Sleep 300ms
Type "agent-deck -p search-demo"
Sleep 400ms
Enter

# Wait for Global Search index to load in background
Sleep 4s

# Navigate to session
Type "j"
Sleep 800ms

# ============================================
# ACT 2: Enter session to show Claude's response
# ============================================
Enter
Sleep 2s

# Scroll up to show the conversation
Ctrl+U
Sleep 2s

# Detach back to dashboard
Ctrl+Q
Sleep 1.5s

# ============================================
# ACT 3: DELETE the session (the "oops" moment)
# ============================================
Sleep 500ms

# Press 'd' to delete
Type "d"
Sleep 800ms

# Confirm deletion with 'y'
Type "y"
Sleep 1.5s

# Session is GONE - show empty dashboard
Sleep 2s

# ============================================
# ACT 4: Open Search with / then switch to Global
# ============================================
Sleep 500ms

# Press / to open search
Type "/"
Sleep 1s

# Press Tab to switch to Global Search
Tab
Sleep 2s

# ============================================
# ACT 5: Search for content from the conversation
# ============================================
Type "JWT"
Sleep 3s

# Results should appear - conversation was found!
Sleep 2s

# Navigate results if multiple
Down
Sleep 1s

# ============================================
# ACT 6: Preview with keyword highlighting
# ============================================

# Scroll preview to show highlighted matches
Type "]"
Sleep 500ms
Type "]"
Sleep 500ms
Type "]"
Sleep 1s

# Scroll back
Type "["
Sleep 500ms
Type "["
Sleep 1s

# Go back to first result
Up
Sleep 800ms

# ============================================
# ACT 7: RECOVER - Press Enter to resume
# ============================================
Enter
Sleep 5s

# ============================================
# FINALE: Session is back! (will be cut in editing)
# ============================================

# Scroll to show the recovered conversation
Ctrl+U
Sleep 2s

# Continue the conversation to prove it works
Type "Can you show me a code example?"
Sleep 300ms
Enter

# Wait for response (will be cut in post)
Sleep 30s

# Scroll up to show response
Ctrl+U
Sleep 2s

# Detach
Ctrl+Q
Sleep 1.5s

# Final view
Sleep 2s
