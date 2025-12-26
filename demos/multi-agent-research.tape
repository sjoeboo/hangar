# Agent Deck - Multi-Agent Research Demo
# Shows: Creating 3 sessions (Claude, Gemini, Codex) for parallel research

Output multi-agent-research.gif
Output multi-agent-research.mp4
Output multi-agent-research.webm

# Terminal settings - Professional styling
Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 16
Set Width 1400
Set Height 900
Set TypingSpeed 30ms
Set Padding 20
Set Margin 20
Set MarginFill "#1a1b26"
Set BorderRadius 10
Set WindowBar Colorful
Set WindowBarSize 40
Set Framerate 30
Set PlaybackSpeed 1.2

# ============================================
# CLEANUP: Remove demo profile sessions (hidden)
# ============================================
Hide
Type "rm -rf ~/.agent-deck/profiles/demo"
Enter
Sleep 300ms
Show

# ============================================
# LAUNCH: Start Agent Deck (clean demo profile)
# ============================================
Sleep 300ms
Type "agent-deck -p demo"
Sleep 300ms
Enter
Sleep 2s

# ============================================
# CREATE SESSION 1: Claude - API Backend
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 200ms

Type "API Backend"
Sleep 200ms
Tab
Sleep 200ms
Type "~/projects/backend"
Sleep 200ms
Tab
Sleep 300ms

# Claude is default, just create
Enter
Sleep 1.5s

# ============================================
# CREATE SESSION 2: Gemini - Web Frontend
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 200ms

Type "Web Frontend"
Sleep 200ms
Tab
Sleep 200ms
Type "~/projects/frontend"
Sleep 200ms
Tab
Sleep 200ms

# Select Gemini (1 Right)
Right
Sleep 300ms

Enter
Sleep 1.5s

# ============================================
# CREATE SESSION 3: Codex - Database Logic
# ============================================
Type "n"
Wait+Screen /New Session/
Sleep 200ms

Type "Database Logic"
Sleep 200ms
Tab
Sleep 200ms
Type "~/projects/database"
Sleep 200ms
Tab
Sleep 200ms

# Select Codex (3 Rights: claude -> gemini -> opencode -> codex)
Right
Sleep 100ms
Right
Sleep 100ms
Right
Sleep 300ms

Enter
Sleep 1.5s

# ============================================
# NOW WRITE IN SESSIONS (Claude had time to start)
# ============================================

# Go to Claude session (first session - 2 Ups from Codex)
# Cursor is on Database Logic (Codex)
# Up -> Web Frontend (Gemini)
# Up -> API Backend (Claude)
Up
Sleep 500ms
Up
Sleep 800ms

# Pause to show we selected Claude (first session)
Sleep 1s

# Enter Claude
Enter
Sleep 2s

# Write in Claude
Type "Explain API authentication patterns"
Sleep 300ms
Enter
Sleep 3s

# Detach
Ctrl+Q
Sleep 1s

# Navigate to Gemini (second session)
Down
Sleep 800ms

# Pause to show we selected Gemini
Sleep 500ms

# Enter Gemini
Enter
Sleep 2s

# Write in Gemini
Type "Compare state management options"
Sleep 300ms
Enter
Sleep 3s

# Detach
Ctrl+Q
Sleep 1s

# Navigate to Codex (third session)
Down
Sleep 800ms

# Pause to show we selected Codex
Sleep 500ms

# Enter Codex
Enter
Sleep 2s

# Write in Codex
Type "Optimize database queries"
Sleep 300ms
Enter
Sleep 3s

# Detach
Ctrl+Q
Sleep 1s

# ============================================
# FINALE: Show all sessions
# ============================================

# Pause to show dashboard with all 3 sessions
Sleep 2s

# Navigate back to Claude (first session)
Up
Sleep 500ms
Up
Sleep 800ms

# Pause to show we selected Claude again
Sleep 1.5s

# Enter Claude to end
Enter
Sleep 3s
