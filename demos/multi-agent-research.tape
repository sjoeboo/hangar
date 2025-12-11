# Agent Deck - Multi-Agent Research Demo
# Shows: Creating 3 sessions (Claude, Gemini, Codex) for parallel research

Output demos/multi-agent-research.gif
Output demos/multi-agent-research.mp4
Output demos/multi-agent-research.webm

# Terminal settings
Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 14
Set Width 1400
Set Height 900
Set TypingSpeed 50ms
Set Padding 20
Set WindowBar "Colorful"
Set BorderRadius 10
Set Framerate 30

# ============================================
# CLEANUP: Remove demo profile sessions (hidden)
# ============================================
Hide
Type "rm -rf ~/.agent-deck/profiles/demo"
Enter
Sleep 500ms
Show

# ============================================
# LAUNCH: Start Agent Deck (clean demo profile)
# ============================================
Sleep 500ms
Type "agent-deck -p demo"
Sleep 500ms
Enter
Sleep 3s

# ============================================
# SESSION 1: Claude - API Backend
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 300ms

# Session name
Type "API Backend"
Sleep 400ms

# Tab to Path
Tab
Sleep 400ms

# Path
Type "~/projects/backend"
Sleep 400ms

# Tab to Command
Tab
Sleep 400ms

# Select Claude (Right once: shell -> claude)
Right
Sleep 600ms

# Create session
Enter
Sleep 2s

# Navigate down to session, then attach
Down
Sleep 300ms
Enter
Sleep 2s

# Type task quickly
Type "Explain API authentication patterns"
Sleep 400ms
Enter
Sleep 3s

# Detach back to dashboard
Ctrl+Q
Sleep 1.5s

# ============================================
# SESSION 2: Gemini - Web Frontend
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 300ms

# Session name
Type "Web Frontend"
Sleep 400ms

# Tab to Path
Tab
Sleep 400ms

# Path
Type "~/projects/frontend"
Sleep 400ms

# Tab to Command
Tab
Sleep 400ms

# Select Gemini (Right twice)
Right
Sleep 300ms
Right
Sleep 600ms

# Create session
Enter
Sleep 2s

# Navigate down to session, then attach
Down
Sleep 300ms
Enter
Sleep 2s

# Type task quickly
Type "Compare state management options"
Sleep 400ms
Enter
Sleep 3s

# Detach back to dashboard
Ctrl+Q
Sleep 1.5s

# ============================================
# SESSION 3: Codex - Backend Logic
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 300ms

# Session name
Type "Backend Logic"
Sleep 400ms

# Tab to Path
Tab
Sleep 400ms

# Path
Type "~/projects/database"
Sleep 400ms

# Tab to Command
Tab
Sleep 400ms

# Select Codex (Right 4 times)
Right
Sleep 200ms
Right
Sleep 200ms
Right
Sleep 200ms
Right
Sleep 600ms

# Create session
Enter
Sleep 2s

# Navigate down to session, then attach
Down
Sleep 300ms
Enter
Sleep 2s

# Type task quickly
Type "Optimize database queries"
Sleep 400ms
Enter
Sleep 3s

# Detach back to dashboard
Ctrl+Q
Sleep 2s

# ============================================
# FINALE: Show all sessions and enter first
# ============================================

# Go to top to show all 3 sessions
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms

# Pause to show dashboard with all 3
Sleep 2s

# Enter first session
Down
Sleep 300ms
Enter
Sleep 3s
