# Agent Deck - Fork Session Demo
# Shows: Fork a Claude session to branch the conversation

Output fork-session.gif
Output fork-session.mp4
Output fork-session.webm

# Terminal settings
Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 18
Set Width 1400
Set Height 900
Set TypingSpeed 40ms
Set Padding 20
Set WindowBar "Colorful"
Set BorderRadius 10
Set Framerate 30
Set PlaybackSpeed 1.5

# ============================================
# SETUP: Create session with conversation (hidden)
# ============================================
Hide

# Clean demo profile and project
Type "rm -rf ~/.agent-deck/profiles/fork-demo && mkdir -p ~/demo-project"
Enter
Sleep 500ms

# Start agent-deck, create session
Type "agent-deck -p fork-demo"
Enter
Sleep 2s

# Create new session
Type "n"
Sleep 1s
Type "Code Review"
Tab
Type "~/demo-project"
Tab
Enter
Sleep 12s

# Enter the session and wait for Claude to fully start
Type "j"
Sleep 500ms
Enter
Sleep 15s

# Start a conversation
Type "What are the key principles of good code review?"
Sleep 300ms
Enter
Sleep 20s

# Detach to return to dashboard
Ctrl+Q
Sleep 1.5s

# Quit agent-deck (session stays running)
Type "q"
Sleep 1s

Show

# ============================================
# DEMO START: Launch with pre-running session
# ============================================
Sleep 500ms
Type "agent-deck -p fork-demo"
Sleep 400ms
Enter
Sleep 3s

# Select the session
Type "j"
Sleep 1s

# ============================================
# STEP 1: Enter session (already loaded with conversation)
# ============================================
Enter
Sleep 5s

# Scroll up to show the conversation
Ctrl+U
Sleep 2s

# Detach
Ctrl+Q
Sleep 1.5s

# ============================================
# STEP 2: Fork the session with 'f'
# ============================================
Sleep 500ms

# Press 'f' for quick fork
Type "f"
Sleep 8s

# ============================================
# STEP 3: Show the forked session in list
# ============================================
# The forked session appears - navigate to it
Sleep 1s
Type "j"
Sleep 2s

# ============================================
# STEP 4: Enter forked session to show context preserved
# ============================================
Enter
Sleep 6s

# Scroll up to show same conversation context is there
Ctrl+U
Sleep 2s

# Continue the conversation from the fork point
Type "Can you give me a code review checklist?"
Sleep 300ms
Enter
Sleep 6s

# ============================================
# FINALE: Detach and show both sessions
# ============================================
Ctrl+Q
Sleep 1.5s

# Navigate up to show both sessions in list
Type "k"
Sleep 1s

# Final view showing original and forked session
Sleep 2s
