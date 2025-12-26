# Twitter Post Templates for Agent Deck

## Post Format Strategy

Each post should follow this structure:
1. **Hook** (first line) - Problem statement or surprising claim
2. **Demo** (GIF/video) - Visual proof
3. **Context** (2-3 lines) - Why it matters
4. **CTA** (Call to Action) - Link to repo/docs

---

## Template 1: The "Before/After" Pattern

```
Managing 40+ AI coding sessions across projects?

[ATTACH: GIF showing agent-deck TUI with grouped sessions, status indicators]

agent-deck gives you a terminal dashboard for all your Claude/Gemini/OpenCode sessions.

✓ See what's running/waiting at a glance
✓ Group by project
✓ Attach/detach without losing context

github.com/asheshgoplani/agent-deck
```

**Visual to capture:**
- Record agent-deck TUI showing 20+ sessions organized in groups
- Navigate between sessions with keyboard shortcuts
- Show status indicators changing (green/yellow/gray dots)
- End with attaching to a Claude session

**Tools:** `asciinema record`, then convert to GIF with `agg` or `terminalizer`

---

## Template 2: The "Single Feature Spotlight"

```
You know what's annoying?

Losing track of which AI agent is waiting for your input.

[ATTACH: Short video showing status transitions]

agent-deck solves this with smart status detection:
● Green = actively working
◐ Yellow = waiting for YOU
○ Gray = idle

Never miss a prompt again.

Try it: curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash
```

**Visual to capture:**
- Split screen: Claude session on left, agent-deck on right
- Trigger a Claude prompt
- Watch status change from green → yellow in real-time
- Show attaching to the waiting session

---

## Template 3: The "Power User Thread" (Multi-tweet)

**Tweet 1:**
```
I manage 30+ concurrent AI coding sessions.

Here's how I stay sane 🧵

[ATTACH: Wide screenshot of agent-deck with multiple groups expanded]
```

**Tweet 2:**
```
1/ Organization

Groups + subgroups = project hierarchy
No more "wait which session was for the API refactor?"

[ATTACH: GIF of creating groups and moving sessions]
```

**Tweet 3:**
```
2/ MCP Management

Attach/detach Model Context Protocol servers per-session
Global or project-local scope

Press 'M' → toggle MCPs → restart session
Claude picks up the new config instantly

[ATTACH: GIF of MCP Manager dialog]
```

**Tweet 4:**
```
3/ Fork & Resume

Fork a Claude session to explore alternative approaches
Original session keeps running

When you find the winner, merge manually

[ATTACH: Short video of forking a session]
```

**Tweet 5:**
```
4/ Global Search

Search across ALL your Claude conversations
Recent work surfaces first
Jump directly to any session

[ATTACH: GIF of Global Search in action]
```

**Tweet 6:**
```
Free & open source
macOS + Linux + WSL
Built with Go + Bubble Tea

github.com/asheshgoplani/agent-deck

⭐ if you find it useful
```

---

## Template 4: The "Problem Agitation"

```
Switching between 10 Claude sessions in tmux?

Manually typing session names?

Forgetting which one has the error you need to fix?

[ATTACH: Meme-style GIF of frustration OR smooth demo]

agent-deck: keyboard-driven session manager for AI coding agents

Press 'j/k' to navigate, 'Enter' to attach, '/' to search

github.com/asheshgoplani/agent-deck
```

**Visual options:**
- Option A: Screen recording showing clunky `tmux attach -t agentdeck_...` workflow
- Option B: Smooth agent-deck navigation with fuzzy search

---

## Template 5: The "Stats/Numbers Hook"

```
160 tmux subprocess spawns per second

That's what duplicate agent-deck instances caused last week

[ATTACH: Screenshot of Activity Monitor showing CPU spike OR htop]

Fixed with PID locking + ghost session optimization

Your terminal shouldn't burn your laptop 🔥

Open source + detailed postmortem in the README:
github.com/asheshgoplani/agent-deck
```

**Visual:**
- Before: Activity Monitor showing high CPU
- After: Calm CPU graph with agent-deck running
- OR: Terminal showing `ps aux | grep tmux` with hundreds of processes

---

## Template 6: The "Built in Public" Story

```
Built a TUI session manager for AI coding agents

Started with "I have too many Claude sessions"

Now includes:
- MCP server management
- Session forking
- Global conversation search
- Performance optimizations (97% subprocess reduction)

[ATTACH: Evolution GIF showing v0.1 → v0.5.8 features]

Building in public is wild. Here's the journey:
github.com/asheshgoplani/agent-deck
```

---

## Template 7: The "Quick Demo" (Video-first)

```
agent-deck in 30 seconds

[ATTACH: 30-second video with captions]

Terminal session manager for Claude, Gemini, OpenCode, etc.

github.com/asheshgoplani/agent-deck
```

**Video script:**
1. Show 20+ sessions in groups (3s)
2. Filter by status with `/` search (3s)
3. Open MCP Manager, toggle servers (5s)
4. Fork a Claude session (4s)
5. Global Search across conversations (5s)
6. Attach to session, show live status updates (5s)
7. Show logo + GitHub link (5s)

**Captions on screen:**
- "Manage 30+ AI sessions"
- "Keyboard-driven navigation"
- "MCP server management"
- "Fork & resume sessions"
- "Search all conversations"
- "Live status indicators"

---

## Media Creation Guide

### Tools for GIF/Video Capture

**Terminal Recording:**
```bash
# 1. Record with asciinema (best quality)
asciinema rec agent-deck-demo.cast

# 2. Convert to GIF
npm install -g @asciinema/agg
agg agent-deck-demo.cast agent-deck-demo.gif

# OR use terminalizer
npm install -g terminalizer
terminalizer record demo
terminalizer render demo
```

**Screen Recording (macOS):**
- **QuickTime Player**: File → New Screen Recording
- **Cmd+Shift+5**: Native screenshot toolbar
- **OBS Studio**: For advanced editing

**GIF Optimization:**
```bash
# Install gifski for high-quality GIF conversion
brew install gifski

# Convert video to GIF
gifski -o output.gif --width 800 --fps 15 input.mov

# Or use ffmpeg
ffmpeg -i input.mov -vf "fps=15,scale=800:-1:flags=lanczos" -loop 0 output.gif
```

### Recording Tips

1. **Clean terminal setup:**
   - Use Tokyo Night theme (matches agent-deck colors)
   - Font size 16-18pt for readability
   - Clear prompt (`PS1='$ '`)
   - Remove distracting background processes

2. **Demo script:**
   - Write out commands beforehand
   - Use `sleep 2` between actions for pacing
   - Reset to clean state before recording

3. **Timing:**
   - GIFs: 5-15 seconds max
   - Videos: 30-60 seconds ideal
   - Loops: Make GIFs loop seamlessly

4. **File size:**
   - Twitter max: 5MB for GIFs
   - Compress with `gifsicle` if needed:
     ```bash
     brew install gifsicle
     gifsicle -O3 --lossy=80 -o optimized.gif input.gif
     ```

---

## Visual Content Ideas

### GIFs to Create

1. **Navigation demo** (10s)
   - j/k to move through sessions
   - Tab to expand/collapse groups
   - Enter to attach
   - Loop seamlessly

2. **Status transitions** (8s)
   - Show session going: idle → running → waiting
   - Highlight color changes

3. **MCP Manager** (12s)
   - Press 'M'
   - Navigate with arrows
   - Toggle spaces to enable/disable
   - Switch LOCAL/GLOBAL scope
   - Apply changes

4. **Fork & resume** (15s)
   - Select a running Claude session
   - Press 'F'
   - Show dialog with options
   - New session spawns
   - Both running in parallel

5. **Global Search** (10s)
   - Press 'G'
   - Type search query
   - Results appear with highlights
   - Preview scrolls to match
   - Jump to session

6. **Quick filters** (8s)
   - Press '!' for running only
   - Press '@' for waiting only
   - Press '0' for all
   - Show session list changing

### Screenshots to Create

1. **Main TUI overview**
   - Full screen with 30+ sessions
   - Multiple groups expanded
   - Various status indicators
   - Help bar visible

2. **Group hierarchy**
   - Deep nesting (3-4 levels)
   - Show project organization

3. **MCP Manager dialog**
   - Both columns visible
   - Mix of enabled/disabled MCPs
   - LOCAL vs GLOBAL indicators

4. **Help overlay**
   - Press '?' view
   - Shows all keyboard shortcuts

---

## Posting Schedule Strategy

### Week 1: Introduction
- **Monday**: Template 1 (overview with main TUI)
- **Wednesday**: Template 2 (status detection feature)
- **Friday**: Template 7 (30-second video demo)

### Week 2: Deep Dive
- **Monday**: Template 3 Thread (power user features)
- **Thursday**: Template 4 (problem/solution)

### Week 3: Technical
- **Tuesday**: Template 5 (performance stats)
- **Friday**: Template 6 (built in public story)

### Ongoing
- Reply to users asking "how do you manage multiple AI sessions?"
- Quote tweet discussions about AI coding tools
- Share updates/bug fixes as short posts

---

## Engagement Tactics

1. **Reply to relevant threads:**
   - Search: "managing claude sessions", "tmux sessions", "ai coding workflow"
   - Reply with agent-deck solution + GIF

2. **Tag relevant accounts:**
   - @ClaudeAI (when posting Claude-specific features)
   - @bubble_tea_tui (built with Bubble Tea)
   - @golang (built with Go)

3. **Use hashtags sparingly:**
   - #DevTools
   - #TerminalUI
   - #AIcoding
   - Max 2 per post

4. **Engage with comments:**
   - Respond to questions quickly
   - Thank users for stars/feedback
   - Share user-created content

---

## Analytics to Track

- Which format gets most engagement? (GIF vs video vs screenshot)
- Which features resonate? (status detection, MCP management, search)
- Best posting times for your audience
- Conversion rate: impressions → GitHub stars

---

## Content Calendar Template

| Date | Post Type | Media | Status |
|------|-----------|-------|--------|
| 2025-12-24 | Template 7 (video) | 30s demo | Ready to film |
| 2025-12-26 | Template 2 (status) | Status GIF | Need to record |
| 2025-12-28 | Template 1 (overview) | TUI screenshot | Have screenshot |

---

## Quick Reference: Best Practices

✅ **DO:**
- Show real usage (not contrived demos)
- Keep GIFs under 10s when possible
- Add captions to videos
- Use your actual project sessions (more authentic)
- Post during US business hours (9am-5pm PT for dev audience)

❌ **DON'T:**
- Over-polish (developers prefer authentic)
- Use stock photos/generic imagery
- Post too frequently (3-4x/week max)
- Make it all about features (share the story too)

---

## Example Post You Could Do RIGHT NOW

```
Just shipped agent-deck 0.5.8 with session cache optimization

Before: 30 tmux subprocesses per tick
After: 1 batched call with 2s cache

CPU usage for idle sessions: 15% → 0.5%

[ATTACH: GIF of Activity Monitor before/after OR htop comparison]

When managing 30+ AI coding sessions, every optimization counts

github.com/asheshgoplani/agent-deck
```

**Media:** Screenshot comparing `ps aux | grep tmux` output before/after optimization
