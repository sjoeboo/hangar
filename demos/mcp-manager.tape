# Agent Deck - MCP Manager Demo
# Shows: Check /mcp (none) -> Attach MCP in LOCAL -> Check /mcp (attached)

Output mcp-manager.gif
Output mcp-manager.mp4
Output mcp-manager.webm

# Terminal settings - Professional styling
Set Shell "zsh"
Set Theme "TokyoNight"
Set FontSize 22
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
# SETUP: Create and launch session (hidden)
# ============================================
Hide

# Clean demo profile and project
Type "rm -rf ~/.agent-deck/profiles/mcp-demo && mkdir -p ~/demo-project"
Enter
Sleep 500ms

# Remove any existing .mcp.json (start with no MCPs)
Type "rm -f ~/demo-project/.mcp.json"
Enter
Sleep 200ms

# Start agent-deck, create session, and wait for Claude to load
Type "agent-deck -p mcp-demo"
Enter
Sleep 3s

# Create new session
Type "n"
Sleep 1s
Type "Research Assistant"
Tab
Type "~/demo-project"
Tab
Enter
Sleep 12s

# Quit agent-deck (session stays running in tmux)
Type "q"
Sleep 1s

Show

# ============================================
# DEMO START: Launch agent-deck with pre-running session
# ============================================
Sleep 1s
Type "agent-deck -p mcp-demo"
Sleep 200ms
Enter
Sleep 2.5s

# ============================================
# SELECT SESSION
# ============================================
Down
Sleep 800ms

# ============================================
# STEP 1: Enter session and check MCPs (none attached)
# ============================================
Enter
Sleep 2s

# Run /mcp to show no MCPs attached
Type "/mcp"
Sleep 200ms
Enter
Sleep 2.5s

# ============================================
# EXIT: Detach from session (Ctrl+Q)
# ============================================
Ctrl+Q
Sleep 1.5s

# ============================================
# STEP 2: Open MCP Manager and attach in LOCAL
# ============================================
Type "M"
Sleep 2s

# We're in LOCAL scope (default) - navigate to Available column
Right
Sleep 800ms

# Press Space to attach the MCP
Space
Sleep 1s

# ============================================
# APPLY: Press Enter to save and restart session
# ============================================
Sleep 500ms
Enter
Sleep 8s

# ============================================
# STEP 3: Enter session and verify MCP is now attached
# ============================================
Enter
Sleep 2s

# Run /mcp to show the MCP is now attached
Type "/mcp"
Sleep 400ms
Enter
Sleep 3s

# ============================================
# FINALE: Detach and show dashboard
# ============================================
Ctrl+Q
Sleep 1s

# Final view - preview shows MCP attached indicator
Sleep 2s
