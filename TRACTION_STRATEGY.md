# Agent Deck: Complete Traction & Growth Strategy

**Version 0.5.8 Launch Plan**

Based on comprehensive research across Hacker News, Product Hunt, Twitter/X, Reddit, and GitHub trending strategies.

---

## Executive Summary

Agent-deck is a terminal session manager for AI coding agents built with Go + Bubble Tea. With the AI coding tools market exploding (Claude Code, Cursor, OpenCode, Gemini Code Assist), there's a massive opportunity to solve the session management problem.

**Target Audience:** Developers managing 10+ AI coding sessions across projects (estimated 50K+ users based on Claude Code adoption)

**Unique Value Props:**
1. Only tool built specifically for AI agent session management
2. Beautiful Tokyo Night TUI with keyboard-first design
3. MCP server management built-in
4. Data protection (session loss prevention after 37-session incident)
5. Privacy-first (local, no telemetry, MIT license)

---

## 90-Day Launch Roadmap

### Phase 1: Foundation (Days 1-30)

**Week 1-2: Polish & Prepare**
- [ ] Create VHS demo tapes (5 key workflows)
- [ ] Optimize GitHub README with GIFs, badges, better structure
- [ ] Add 20 topics to GitHub repo
- [ ] Enable GitHub Discussions with categories
- [ ] Create public roadmap in Projects
- [ ] Write 3 blog posts (draft stage)
- [ ] Build Reddit comment karma (100+ across target subs)

**Week 3-4: Content Creation**
- [ ] Record 30-second product demo video
- [ ] Create Twitter content calendar (21 posts)
- [ ] Design Product Hunt gallery images (7 images, 2540x1520px)
- [ ] Write Show HN post draft
- [ ] Prepare Reddit launch posts (4 subreddits)
- [ ] Create Twitter thread series (5 threads)

### Phase 2: Soft Launch (Days 31-45)

**Week 5: Initial Validation**
- [ ] Post to r/SideProject (most welcoming)
- [ ] Post to r/coolgithubprojects (easy feedback)
- [ ] Share on personal Twitter
- [ ] Email 10-20 close contacts for feedback
- [ ] Fix obvious bugs and UX issues
- [ ] **Target:** 50-100 GitHub stars

**Week 6: Content Marketing**
- [ ] Publish blog: "Building a TUI with Go + Bubble Tea: Lessons Learned"
- [ ] Cross-post to dev.to and Hashnode
- [ ] Tweet thread: "Building agent-deck: The Journey"
- [ ] Engage in r/commandline TUI discussions
- [ ] Answer questions in r/ClaudeAI
- [ ] **Target:** 100-200 GitHub stars

### Phase 3: Main Launch (Days 46-60)

**Week 7: Reddit + Hacker News**
- [ ] Monday 9am EST: r/commandline launch
- [ ] Monday 3pm EST: r/ClaudeAI launch
- [ ] Tuesday 10am PST: Show HN launch
- [ ] Wednesday: r/golang (if karma sufficient)
- [ ] Monitor and respond to all comments within 2 hours
- [ ] **Target:** 500-1,000 GitHub stars

**Week 8: Product Hunt**
- [ ] Saturday 12:01am PT: Product Hunt launch
- [ ] Coordinate supporter outreach (stagger by timezone)
- [ ] Respond to every comment within 2-hour intervals
- [ ] Cross-promote on Twitter throughout the day
- [ ] **Target:** Top 5 Dev Tool of Day, 500+ PH upvotes

### Phase 4: Amplification (Days 61-90)

**Weeks 9-12: Sustained Growth**
- [ ] Weekly feature releases with announcements
- [ ] Bi-weekly blog posts (technical deep-dives)
- [ ] YouTube tutorial series (5 videos)
- [ ] Reach out to 30 tech influencers
- [ ] Submit to awesome lists (awesome-go-cli, awesome-tuis)
- [ ] Host first Twitter Space on terminal productivity
- [ ] **Target:** 1,500-3,000 GitHub stars

---

## Platform-Specific Strategies

### GitHub: The Foundation

**README Optimization (Critical - Developers decide in 5 seconds)**

**Above the Fold:**
```markdown
# agent-deck
Terminal session manager for AI coding agents

[30-second VHS demo GIF showing session tree → MCP manager → fork]

[![Build Status](badge-url)] [![Go Report Card](badge-url)] [![Release](badge-url)] [![License](badge-url)]

**Quick Install:**
brew install asheshgoplani/tap/agent-deck
```

**Topics to Add (Use all 20 slots):**
```
terminal, tui, session-manager, tmux, go, bubbletea,
developer-tools, productivity, cli, ai-tools, claude-code,
bubble-tea, terminal-ui, golang-cli, developer-productivity,
ai-coding-agents, mcp, model-context-protocol, tokyo-night, gemini
```

**GitHub Features to Enable:**
- [x] Discussions (categories: Announcements, Show & Tell, Q&A, Feature Requests, General)
- [x] Projects (public roadmap)
- [x] Sponsorship (GitHub Sponsors for sustainability)
- [x] Wiki (comprehensive guides)

**Community Building:**
- Pin "Welcome" discussion with setup guide
- Create "Most helpful" contributor recognition
- Post weekly updates (even small ones)
- Label issues as "good first issue" for contributors
- Respond to all issues/PRs within 24 hours

---

### Hacker News: The Developer Audience

**Show HN Format:**
- **Title:** "Show HN: agent-deck - Terminal session manager for AI coding agents"
- **URL:** Direct to GitHub repo (not blog post or landing page)
- **Time:** Tuesday or Wednesday, 10am-12pm PST (optimal for developer audience)

**Comment Strategy (Post First Comment Immediately):**
```markdown
Hi HN! I'm the author of agent-deck.

I built this after losing 37 Claude Code sessions to a tmux server restart.
The pain point: managing 20+ AI coding sessions across projects, losing
track of which agent was waiting for my input, and manually juggling MCP
configurations.

agent-deck gives you a Tokyo Night-themed TUI to manage all your Claude/
Gemini/OpenCode sessions in one place. Key features:

- Smart status detection (running/waiting/idle)
- MCP Manager for attaching Model Context Protocol servers
- Session forking without losing context
- Global search across all conversations
- Data protection (PID locks, automatic backups)

Tech stack: Go + Bubble Tea (TUI framework) + tmux integration

Some interesting technical challenges:
- Subprocess batching (97% reduction from 30 calls/tick → 1)
- Content hashing for stable status detection
- Profile isolation with test safety guards

Happy to answer questions! Would especially love feedback from others
managing multiple AI agent sessions.

Install: brew install asheshgoplani/tap/agent-deck
```

**Engagement Rules:**
- Respond to EVERY comment authentically
- Be humble about limitations
- Share technical details when asked
- Don't argue with critics
- Thank people for feedback
- Fix bugs mentioned in comments immediately

**What NOT to Do on HN:**
- ❌ Use superlatives (fastest, best, revolutionary)
- ❌ Marketing speak or hype
- ❌ Ignore negative feedback
- ❌ Argue defensively
- ❌ Post on weekends (lower traffic)

---

### Product Hunt: The Launch Event

**Pre-Launch Checklist (30 days before)**

**Hunter Selection:**
- Contact experienced Hunter 2-4 weeks before
- They provide external perspective on messaging
- Hunters with track record increase credibility

**Asset Creation:**

**Tagline (60 characters max):**
- ✅ "Manage multiple AI coding sessions in one TUI"
- ✅ "Terminal session manager for Claude/Gemini/OpenCode"
- ❌ "The ultimate terminal productivity tool" (too generic)

**Gallery Images (7 images, 2540x1520px):**
1. Opening: "Juggling 40 Claude sessions?" (problem statement)
2. Product screenshot: Full TUI with session tree
3. Feature: MCP Manager dialog
4. Feature: Global Search in action
5. Feature: Session forking workflow
6. Outcome: "20 sessions → 1 interface" transformation
7. CTA: Installation command + GitHub link

**Tools:** Hotpot.ai Product Hunt Gallery Generator, Brandbird, Screenhance

**Video (3-5 minutes):**
- Loom recording showing full workflow
- Start → create session → attach MCP → fork → search
- Upload to YouTube (public, not unlisted)

**Maker's Comment (800 characters):**
```markdown
I built agent-deck after juggling 30+ Claude Code sessions across multiple
projects. Switching between tmux sessions, losing track of which agent was
working on what, and manually managing MCPs became overwhelming.

agent-deck is a terminal session manager built specifically for AI coding
agents. It gives you a Tokyo Night-themed TUI to manage all your Claude/
Gemini/OpenCode sessions in one place, with built-in MCP management and
session forking.

Key features:
• Real-time status indicators (running/waiting/idle)
• MCP server attachment (local or global scope)
• Fork sessions without losing context
• Global search across all conversations
• Data protection (learned the hard way after losing 37 sessions)

Try it: brew install asheshgoplani/tap/agent-deck

Would love your feedback on the MCP management workflow - it's the most
unique feature and I'm curious if it resonates with Claude Code users!
```

**Launch Day Strategy:**

**Timing:** Saturday or Sunday (less competition, higher developer ratio)

**Supporter Coordination:**
- Email first wave at 12:01am PT (launch time)
- Stagger outreach by timezone (Pacific → Eastern → Europe → Asia)
- Send 5-7 waves throughout 24 hours
- Aim for 25-50 votes/hour (appears organic to algorithm)

**Team Roles:**
- Person 1: Own PH thread, respond to comments (2-hour intervals)
- Person 2: Monitor traffic, signups, activation metrics
- Person 3: Manage social posts (Twitter, LinkedIn)
- Person 4: Handle product issues, demo stability

**Cross-Promotion Schedule:**
- 12:01am PT: Post to Twitter
- 7:00am PT: Email East Coast supporters
- 2:00pm PT: Third push to maintain momentum
- Evening: Continue responding to comments

**Success Metrics:**
- Top 5 Dev Tool of Day (minimum goal)
- 500+ upvotes (realistic with preparation)
- 1,500 unique visitors (benchmark for Top 4)
- 100-300 signups/installs (B2B tool benchmark)

---

### Twitter/X: The Community Builder

**Content Strategy: 70% Value, 30% Promotion**

**Weekly Posting Schedule:**
- Monday: Technical insight or challenge solved
- Tuesday: Build in public update
- Wednesday: Engagement (poll, question, retweet)
- Thursday: Feature highlight (GIF)
- Friday: Community spotlight (user showcase)
- Weekend: Personal story or casual content

**Launch Campaign (3 Weeks)**

**Week 1: Teaser Phase**
```
Managing multiple AI coding agents is chaos.

15 Claude sessions open.
4 Cursor instances.
3 OpenCode terminals.

Which one has the error you need to fix?

We're solving this. Stay tuned. [Screenshot of messy terminal windows]
```

**Week 2: Educational Phase**
```
We analyzed 50 developer workflows managing AI coding sessions.

Common pain points:
• Lost track of which agent was waiting
• Manual MCP server configuration per session
• Session crashes = lost context
• No visual overview of all running agents

Building the solution. Beta launches next week.

[GIF of agent-deck prototype]
```

**Week 3: Launch Thread**
```
(1/7) Introducing agent-deck: Terminal session manager for AI coding agents

Built after losing 37 Claude Code sessions to a tmux restart.

Never lose context again. [30-second demo video]

(2/7) Problem: You're managing 10+ AI agent sessions across projects.

Claude for backend, Cursor for frontend, OpenCode for scripts.

Which one was working on the auth bug? Which one's waiting for you?

[GIF showing chaos vs. organized session tree]

(3/7) Solution: One TUI to manage them all

• Visual session tree with groups
• Real-time status (● running | ◐ waiting | ○ idle)
• Attach/detach with keyboard shortcuts
• Fork sessions without losing context

[GIF of navigation]

(4/7) Built-in MCP Manager

Attach Model Context Protocol servers per-session.

Toggle on/off with spacebar. Choose LOCAL or GLOBAL scope.

Press 'M' → Select MCPs → Restart session → Claude picks up new config

[GIF of MCP Manager]

(5/7) Global Search

Search across ALL your Claude conversations.

Find that solution you discussed 3 days ago.

Jump directly to any session.

[GIF of search in action]

(6/7) Data Protection

After the 37-session incident, we built:
• PID locks (prevent duplicate instances)
• Ghost session optimization
• Automatic log cleanup
• Rolling backups

Your work is safe.

(7/7) Try it now:

brew install asheshgoplani/tap/agent-deck

Or: curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/install.sh | bash

GitHub: https://github.com/asheshgoplani/agent-deck

Built with Go + Bubble Tea. MIT licensed. No telemetry.

⭐ if you find it useful!
```

**Hashtag Strategy (2-3 per tweet):**
- #BuildInPublic (primary - community movement)
- #DevTools (category)
- #TerminalUI or #TUI (niche)
- #AI or #AIcoding (when relevant)
- #Golang or #Go (tech stack)

**Accounts to Tag (When Relevant):**
- @charmcli (Bubble Tea framework)
- @golang (Go language)
- @ClaudeAI (when posting Claude-specific features)

**Engagement Tactics:**
- Reply to "How do you manage AI sessions?" threads
- Quote tweet AI coding tool discussions
- Host Twitter Space: "Building Production TUIs with Go"
- Celebrate GitHub star milestones (every 100 stars)

**Content Types to Create:**
1. **VHS Demos** (10-15 seconds)
   - Navigation workflow
   - Status transitions
   - MCP Manager
   - Fork & resume
   - Global Search

2. **Technical Threads** (5-7 tweets)
   - "How we reduced CPU usage 97%"
   - "Building with Bubble Tea: Lessons Learned"
   - "tmux Integration Challenges"

3. **Build in Public Updates**
   - Weekly feature announcements
   - Bug fixes with explanations
   - Community feedback integration
   - GitHub star milestones

---

### Reddit: The Developer Communities

**Target Subreddits (In Launch Order)**

**Phase 1: Soft Launch (Week 5)**
1. r/SideProject (most welcoming, encouraged self-promotion)
2. r/coolgithubprojects (easy feedback, low barrier)

**Phase 2: Main Launch (Week 7)**
3. r/commandline (110K subscribers, perfect fit)
4. r/ClaudeAI (408K subscribers, target audience)

**Phase 3: Expansion (Week 9+)**
5. r/golang (339K subscribers, once karma built)
6. r/opensource (210K subscribers, emphasize MIT license)
7. r/linux (general audience)
8. r/tmux (niche community)

**Posting Template for r/commandline**

**Title:** "I built a TUI session manager for AI coding agents (Claude/OpenCode/Cursor)"

**Body:**
```markdown
I was juggling 15+ Claude Code sessions across different projects and kept
losing track of which sessions were waiting for responses. agent-deck solves this.

**What it is:** A terminal session manager built specifically for AI coding
agents. Think tmux + beautiful TUI interface with smart status detection.

**Key features:**
- Visual session tree with groups/subgroups
- Real-time status indicators (running/waiting/idle)
- MCP Manager for attaching servers to Claude sessions
- Global search across all conversations
- Fork sessions without losing context
- Built-in data protection (session loss prevention)

[30-second VHS demo GIF]

**Tech stack:** Go + Bubble Tea (TUI framework) + tmux integration

Built this after losing 37 sessions to a tmux server restart. Now includes
PID locks, ghost session optimization, and automatic log cleanup.

**Install:**
brew install asheshgoplani/tap/agent-deck

Or see GitHub for other methods: https://github.com/asheshgoplani/agent-deck

**Feedback welcome!** Especially interested in hearing from others managing
multiple AI coding sessions.
```

**Posting Template for r/ClaudeAI**

**Title:** "Built a session manager for juggling multiple Claude Code sessions"

**Flair:** "Built with Claude" or "Productivity"

**Body:**
```markdown
Anyone else managing 10+ Claude Code sessions and losing track of which
one's waiting for you?

I built agent-deck to solve this. It's a terminal UI that shows all your
Claude (and other AI agent) sessions in one place with real-time status.

**Why it's useful for Claude users:**
- See at a glance which sessions are waiting for responses
- MCP Manager: attach/detach Model Context Protocol servers per session
- Fork sessions to explore alternative approaches
- Global search across all your Claude conversations
- Never lose context from tmux crashes

[GIF showing session tree with Claude sessions]

**MCP Management** is the killer feature - press 'M' in any session,
toggle MCPs on/off, choose LOCAL or GLOBAL scope, and restart. Claude
picks up the new config instantly.

Built with Go + Bubble Tea. MIT licensed, no telemetry.

GitHub: https://github.com/asheshgoplani/agent-deck
Install: brew install asheshgoplani/tap/agent-deck

Would love feedback from heavy Claude Code users!
```

**Reddit Engagement Rules:**
- Respond to every comment within 2 hours
- Be honest about limitations
- Thank people for suggestions
- Don't argue with critics (acknowledge and learn)
- Follow 90/10 rule (90% participation, 10% self-promo)

---

## Content Creation Checklist

### VHS Demo Tapes (5 Essential Demos)

**1. Quick Navigation (10s)**
- j/k to move through sessions
- Tab to expand/collapse groups
- Enter to attach
- Loop seamlessly

**2. Status Transitions (8s)**
- Show session: idle → running → waiting
- Highlight color changes (green/yellow/gray)
- Demonstrate "never miss a prompt"

**3. MCP Manager (12s)**
- Press 'M' to open
- Navigate with arrows
- Toggle with Space
- Switch LOCAL/GLOBAL scope
- Apply changes, show session restart

**4. Fork & Resume (15s)**
- Select running Claude session
- Press 'F' for fork dialog
- New session spawns with same context
- Both sessions running in parallel
- Show in session tree

**5. Global Search (10s)**
- Press 'G'
- Type search query ("database migration")
- Results appear with highlights
- Preview scrolls to match
- Jump to session

**Tools:**
```bash
# Install VHS
brew install charmbracelet/tap/vhs

# Create tape file
cat > demo.tape <<'EOF'
Output demo.gif
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Tokyo Night"

Type "agent-deck"
Enter
Sleep 2s
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Enter
Sleep 2s
Ctrl+Q
Sleep 1s
EOF

# Generate GIF
vhs demo.tape

# Optimize
gifsicle -O3 --lossy=80 -o demo-optimized.gif demo.gif
```

### Blog Posts (3 Required)

**1. Technical Deep-Dive**
**Title:** "Building a Production TUI with Go + Bubble Tea: Lessons Learned"
**Length:** 2,000-2,500 words
**Sections:**
- Why we chose Bubble Tea over alternatives
- Architecture: Model-Update-View pattern
- tmux integration challenges
- Content hashing for status detection
- Performance optimization (subprocess batching)
- Testing TUI applications
**Publish:** Personal blog + dev.to + Hashnode
**Timing:** Week 6 (before main launch)

**2. Problem/Solution Story**
**Title:** "How I Lost 37 Claude Code Sessions (And Built a Solution)"
**Length:** 1,500-2,000 words
**Sections:**
- The incident (tmux server restart)
- Why existing tools weren't enough
- Designing for data protection
- PID locks and ghost session optimization
- Lessons for building resilient tools
**Publish:** Medium + dev.to + HN as comment link
**Timing:** Week 7 (launch week)

**3. Use Case Guide**
**Title:** "Managing Multiple AI Coding Agents: A Productivity Guide"
**Length:** 1,800-2,200 words
**Sections:**
- When to use multiple agents
- Organizing sessions by project
- MCP server strategies
- Workflow examples (backend/frontend/scripts)
- Keyboard shortcuts for power users
**Publish:** Dev.to + personal blog + r/ClaudeAI
**Timing:** Week 9 (post-launch)

### Videos (2 Required, 3 Optional)

**Required:**

**1. Product Demo (30 seconds)**
- Shot-by-shot storyboard:
  - 0-5s: Problem (show chaotic terminal windows)
  - 6-15s: Solution (clean agent-deck interface)
  - 16-25s: Key features (session tree, MCP manager, search)
  - 26-30s: CTA (GitHub link + install command)
- Platform: Twitter, Product Hunt hero video
- Tool: QuickTime + iMovie for editing

**2. Full Tutorial (10-15 minutes)**
- Introduction (1min): What is agent-deck?
- Installation (2min): All methods (Homebrew, curl, Go install)
- Basic usage (3min): Create, start, attach, detach
- Advanced features (4min): Groups, MCP Manager, search, fork
- Configuration (2min): config.toml, custom tools
- Tips & tricks (2min): Keyboard shortcuts, productivity hacks
- Q&A/FAQ (2min): Common questions
- Platform: YouTube, embedded in README
- Tool: OBS Studio for screen recording

**Optional:**

**3. MCP Manager Deep-Dive (5 minutes)**
- What are MCPs?
- Configuring in ~/.agent-deck/config.toml
- Attaching to sessions (LOCAL vs GLOBAL)
- Use cases (search, databases, APIs)

**4. Developer Walkthrough (20 minutes)**
- Code tour of agent-deck
- Bubble Tea architecture
- tmux integration
- Contribution guide

**5. Performance Optimization Case Study (8 minutes)**
- Before: 30 subprocesses/tick
- After: 1 batched call
- Code walkthrough
- Metrics and benchmarks

---

## Influencer Outreach Strategy

### Target Influencers (30 Total)

**Tier 1: Terminal Tool Creators (10)**
- Jesse Duffield (lazygit, lazydocker)
- Charm Bracelet team (Bubble Tea maintainers)
- btop creator (aristocratos)
- k9s creator
- lazydocker maintainer
- Alacritty maintainers
- Warp team (modern terminal)
- Fig team (terminal productivity)
- Termius developers
- Hyper terminal team

**Tier 2: Go Community Leaders (10)**
- Mat Ryer (Go Time podcast)
- Filippo Valsorda (cryptography, Go)
- Dave Cheney (Go performance)
- Jaana Dogan (observability, distributed systems)
- Natalie Pistunovich (Go community organizer)
- William Kennedy (Ardan Labs)
- Jon Calhoun (Gophercises)
- Todd McLeod (Go courses)
- Alex Ellis (Go, OpenFaaS)
- Caleb Doxsey (Go books)

**Tier 3: AI Coding Tool Reviewers (10)**
- Tech YouTubers reviewing Claude Code
- Cursor power users
- OpenCode contributors
- AI coding tool comparison bloggers
- Developer productivity content creators

### Outreach Template

**Subject:** "Built a terminal session manager for AI coding agents - would love your thoughts"

**Body:**
```
Hi [Name],

I'm Ashesh, and I built agent-deck, a terminal session manager specifically
for managing multiple AI coding agents (Claude, Cursor, Aider, etc.).

Given your work with [their project - e.g., "lazygit and the amazing TUI
you've built"], I thought you might find it interesting. It uses Bubble Tea
for the TUI and solves the problem of juggling 10+ AI agent sessions across
projects.

Key features:
- Visual session tree with real-time status
- MCP server management built-in
- Session forking without losing context

No pressure at all, but if you have a moment to check it out, I'd genuinely
appreciate any feedback: https://github.com/asheshgoplani/agent-deck

Thanks for all your contributions to the terminal tools community!

Best,
Ashesh
```

**Timing:**
- Week 6: Tier 1 (terminal tool creators)
- Week 8: Tier 2 (Go community)
- Week 10: Tier 3 (AI tool reviewers)

**Follow-Up:**
- If no response in 2 weeks, one gentle follow-up
- If they respond positively, offer early access to new features
- If they share publicly, thank them and amplify

---

## Measurement & Analytics

### Key Metrics Dashboard

**GitHub Metrics:**
- Stars over time (use Star History: https://star-history.com/)
- Daily/weekly clone count
- Traffic sources (Insights → Traffic)
- Issue response time (target: <24h)
- PR merge rate
- Contributor count

**Installation Metrics:**
- Homebrew downloads (analytics via tap stats)
- Go install downloads (proxy.golang.org insights)
- Docker pulls (if published)
- Installation method breakdown

**Community Health:**
- GitHub Discussions activity
- Discord/Slack members (if created)
- Reddit post engagement rates
- Twitter follower growth
- YouTube video watch time

**Content Performance:**
- Blog post views (Google Analytics)
- Video completion rates (YouTube Analytics)
- Social media engagement rates:
  - Twitter: (Likes + RTs + Replies) / Followers × 100
  - Reddit: Upvote ratio, comment count
  - HN: Points, comment count

**Conversion Funnel:**
- Product Hunt visitors → GitHub
- GitHub visitors → Installation
- Installation → Active usage (if telemetry added with opt-in)
- Users → Contributors

### Growth Expectations (Realistic Benchmarks)

**Month 1:**
- GitHub Stars: 50-200
- Reddit Combined Karma: 100-300
- Twitter Followers: 50-100
- Homebrew Installs: 100-300

**Month 2:**
- GitHub Stars: 200-1,000 (if trending once)
- HN Front Page: 100-500 points
- Product Hunt: Top 5 Dev Tool
- YouTube Views: 1,000-3,000

**Month 3:**
- GitHub Stars: 1,000-3,000
- Contributors: 5-10
- Weekly Active Installs: 200-500
- Community: 100-200 engaged users

**Month 6:**
- GitHub Stars: 3,000-10,000
- Contributors: 15-30
- Weekly Active Installs: 500-1,000
- Established community (Discord/Discussions)

**Critical Insight:**
> Growth is not linear. You'll have spikes (trending, features) and valleys.
> Focus on sustained engagement over vanity metrics. 100 active users who
> love your tool > 10,000 stars from drive-by clicks.

### Weekly Review Checklist

**Every Monday:**
- [ ] Review GitHub stars trend (up/down/flat)
- [ ] Check issues/PRs (respond to all)
- [ ] Analyze traffic sources (which channels working?)
- [ ] Read all feedback (Reddit, HN, Twitter, GitHub)
- [ ] Plan week's content (1 blog post OR 1 video OR 3 tweets)

**Every Month:**
- [ ] Publish retrospective (what worked, what didn't)
- [ ] Survey top 10 users (feedback call)
- [ ] Update roadmap based on feedback
- [ ] Celebrate milestone (share metrics publicly)
- [ ] Plan next month's features

---

## Common Pitfalls to Avoid

### ❌ DON'T

**1. Launch Without Polishing:**
- Don't post with incomplete README
- Don't launch with obvious bugs
- Don't skip the demo GIF
- First impressions matter - you get one shot

**2. Spam Multiple Platforms Simultaneously:**
- Don't post to 10 subreddits in one day
- Don't tweet 20 times on launch day
- Stagger releases, appear authentic

**3. Ignore Negative Feedback:**
- Don't argue with critics
- Don't get defensive
- Don't dismiss legitimate concerns
- Address or acknowledge all feedback

**4. Oversell / Hype:**
- Don't use superlatives ("revolutionary", "game-changing")
- Don't compare to established tools negatively
- Let the product speak for itself
- Be humble about limitations

**5. Abandon After Launch:**
- Don't disappear post-launch
- Don't let issues go stale (>48h kills trust)
- Don't stop engaging with community
- Consistency > one-time viral spike

**6. Focus Only on Metrics:**
- Don't chase stars over user satisfaction
- Don't optimize for HN points over product quality
- Don't ignore the 10 power users for 1,000 casual stargazers
- Build for users, not for launch day

### ✅ DO

**1. Build in Public:**
- Share progress, challenges, wins
- Post weekly updates (even small ones)
- Be transparent about decisions
- Show the human behind the code

**2. Engage Authentically:**
- Respond to every comment
- Help other developers
- Support fellow builders
- Be part of the community, not just promoting

**3. Iterate Based on Feedback:**
- Implement user suggestions
- Fix bugs immediately
- Show responsiveness
- Credit contributors publicly

**4. Focus on Quality:**
- Polish the core experience
- Make installation trivial
- Write clear documentation
- Ensure demo works perfectly

**5. Play the Long Game:**
- Weekly commits > big releases every 6 months
- Consistent presence > viral spikes
- Community building > one-time launches
- Sustainable growth > hockey stick

---

## Launch Day Checklist

### T-7 Days (One Week Before)

- [ ] README polished with GIFs, badges, clear structure
- [ ] 5 VHS demo tapes created and optimized
- [ ] Product Hunt gallery images finalized (7 images)
- [ ] Show HN post drafted and reviewed
- [ ] Reddit posts drafted for 4 subreddits
- [ ] Twitter thread (7 tweets) scheduled
- [ ] Blog post #1 published (technical deep-dive)
- [ ] 30-second product demo video uploaded to YouTube
- [ ] GitHub Discussions enabled with welcome post
- [ ] Public roadmap created in Projects
- [ ] All bugs from beta testing fixed

### T-1 Day (Day Before Launch)

- [ ] Product Hunt listing submitted (Saturday/Sunday for launch)
- [ ] Hunter confirmed and briefed
- [ ] Supporter outreach email drafted (3 waves by timezone)
- [ ] Social media posts pre-written
- [ ] Demo environment tested (verify all workflows)
- [ ] Analytics set up (Google Analytics, Star History)
- [ ] Team roles assigned (if team launch)
- [ ] Sleep well (seriously - you'll need energy)

### Launch Day (Hour by Hour)

**12:00am PT - Product Hunt Goes Live**
- [ ] Verify listing live on Product Hunt
- [ ] Post to Twitter immediately
- [ ] Send email to Pacific time supporters
- [ ] Pin PH link to Twitter profile

**7:00am PT - East Coast Wakes Up**
- [ ] Send email to East Coast supporters
- [ ] Post to r/commandline (9am EST)
- [ ] Monitor and respond to comments

**10:00am PT - Hacker News Launch**
- [ ] Submit Show HN post
- [ ] Post first comment with context
- [ ] Tweet about HN launch

**12:00pm PT - Midday Push**
- [ ] Post to r/ClaudeAI
- [ ] Check Product Hunt ranking
- [ ] Respond to all comments (PH, HN, Reddit)

**3:00pm PT - Afternoon Momentum**
- [ ] Send third supporter email wave
- [ ] Post to r/golang (if karma sufficient)
- [ ] Share user testimonials on Twitter

**Evening - Sustained Engagement**
- [ ] Respond to all new comments
- [ ] Monitor installation analytics
- [ ] Fix any critical bugs reported
- [ ] Thank supporters publicly

**Before Bed**
- [ ] Schedule next day's follow-up posts
- [ ] Document launch day learnings
- [ ] Get rest for day 2

### T+1 Week (First Week After Launch)

- [ ] Publish blog post #2 (problem/solution story)
- [ ] Post "Thank you" tweet with metrics
- [ ] Respond to all GitHub issues
- [ ] Merge community PRs
- [ ] Plan first feature based on feedback
- [ ] Weekly retrospective (what worked?)

---

## Budget & Resources

### Free Tools (All You Need)

**Content Creation:**
- VHS (terminal demos): Free, open-source
- Asciinema (terminal recording): Free
- gifsicle (GIF optimization): Free
- OBS Studio (screen recording): Free
- Canva Free (graphics): Free tier sufficient

**Platforms:**
- GitHub (hosting): Free for public repos
- Twitter/X: Free
- Reddit: Free
- Hacker News: Free
- Dev.to / Hashnode: Free
- YouTube: Free

**Analytics:**
- GitHub Insights: Built-in
- Star History: Free
- Google Analytics: Free
- Twitter Analytics: Built-in

**Estimated Time Investment:**
- Week 1-4: 20 hours (content creation)
- Week 5-8: 30 hours (launches + engagement)
- Week 9-12: 15 hours/week (sustained growth)
- Total first 90 days: ~150 hours

### Optional Paid Tools

**If Budget Allows:**
- Typefully ($15/mo): Twitter thread scheduling
- Hotpot.ai ($10): Product Hunt gallery creation
- Loom ($12.50/mo): Video recording with captions
- ConvertKit ($29/mo): Email list management for supporters
- Beehiiv ($49/mo): Newsletter for updates

**Recommendation:** Start free, upgrade only if bottlenecks appear

---

## Contingency Plans

### If Hacker News Post Doesn't Gain Traction

**Signs:**
- <10 points after 2 hours
- Not on /newest page
- No comments

**Response:**
- Don't repost immediately (wait 6+ months)
- Focus energy on Reddit and Product Hunt
- Engage in HN comments on related posts
- Build credibility for future launch

### If Product Hunt Launch Underperforms

**Signs:**
- <100 upvotes by end of day
- Not in Top 10

**Response:**
- Don't be discouraged (algorithm changed in 2025)
- Extract value from comments and feedback
- Use PH badge even if not #1
- Focus on GitHub stars as better metric

### If Reddit Posts Get Downvoted

**Signs:**
- <50% upvote ratio
- Negative comments about self-promotion

**Response:**
- Acknowledge feedback gracefully
- Engage with critics respectfully
- Participate more in community before next post
- Focus on value-first content

### If GitHub Stars Grow Slowly

**Signs:**
- <50 stars after 2 weeks
- Low traffic to repo

**Response:**
- Improve README with better GIFs
- Post technical blog content
- Engage in related GitHub repos
- Be patient - quality tools grow slowly

---

## Long-Term Sustainability (Beyond 90 Days)

### Community Building

**Monthly Office Hours:**
- Live call where anyone can ask questions
- Record and publish on YouTube
- Builds personal connections
- Identify power users and potential contributors

**Contributor Recognition:**
- Monthly "Contributor Spotlight" on Twitter
- All Contributors badge in README
- Special Discord role for contributors
- Annual "Top Contributor" award

**Governance:**
- Document decision-making process
- Create GOVERNANCE.md
- Consider project foundations (if large adoption)
- Transparent roadmap

### Monetization (Optional)

**Open Core Model:**
- Core: Free forever (current features)
- Pro ($20-50/month): Priority support, advanced features
- Enterprise ($200+/month): Team features, SSO, SLA

**GitHub Sponsors:**
- Tiers: $5, $20, $100/month
- Rewards: Logo in README, monthly call, feature requests
- Goal: Cover hosting, tool costs

**Key Decision:**
> If it's core to the value proposition, keep it free. If it's about scale,
> reliability, or convenience, charge for it.

### Feature Roadmap

**Q1 2025 (Months 4-6):**
- [ ] Plugin system for custom tools
- [ ] Theme customization
- [ ] Import/export session configurations
- [ ] Remote session support (SSH)

**Q2 2025 (Months 7-9):**
- [ ] Team collaboration features (shared sessions)
- [ ] Session templates
- [ ] Advanced filtering and tagging
- [ ] Web UI (optional companion)

**Q3 2025 (Months 10-12):**
- [ ] AI-powered session suggestions
- [ ] Integration with IDEs (VS Code extension)
- [ ] Performance analytics
- [ ] Enterprise features (SSO, audit logs)

### Content Strategy (Ongoing)

**Monthly Content Plan:**
- 4 blog posts (1/week)
- 2 videos (tutorials or features)
- 20 tweets (3-4/week)
- 1 major feature release
- 1 community highlight

**Topics:**
- User success stories
- Technical deep-dives
- Comparison articles
- Integration guides
- Performance optimizations

---

## Conclusion: The Path to 10K Stars

**Success Factors (In Order of Importance):**

1. **Solve Real Problem** (70% of success)
   - Agent-deck solves genuine pain (managing AI sessions)
   - Target audience is growing (AI coding tools adoption)
   - Unique value prop (MCP management, data protection)

2. **Quality Execution** (20% of success)
   - Beautiful UI (Tokyo Night theme)
   - Thoughtful UX (keyboard-first)
   - Rock-solid reliability (data protection)
   - Clear documentation

3. **Marketing Execution** (10% of success)
   - Multi-platform launch (Reddit, HN, PH, Twitter)
   - Consistent content creation
   - Authentic community engagement
   - Strategic influencer outreach

**The Reality Check:**
- 90% of tools never reach 1,000 stars
- 1% reach 10,000 stars
- Success requires: Great product + Persistence + Luck

**The 90-Day Goal:**
- 1,500-3,000 GitHub stars (realistic)
- 10-20 active contributors
- 500+ weekly active installs
- Established community (Discussions, Discord)
- Foundation for sustainable growth

**Beyond Metrics:**
> The goal isn't just stars. It's building a tool that genuinely helps
> developers. If you solve a real problem and support your users well,
> growth follows naturally.
>
> Focus on the 10 power users who love your tool, not the 10,000 who
> casually starred it. They're your foundation.

**Final Thoughts:**

This is a marathon, not a sprint. lazygit took 5 years to reach 50K+ stars.
Supabase launched 16 times on Product Hunt before finding massive success.

Execute this 90-day plan with discipline and authenticity. Respond to every
comment. Fix every bug. Help every user. Build in public. Stay consistent.

The community will come.

Good luck! 🚀

---

## Quick Reference

**Key Links:**
- Hacker News Guidelines: https://news.ycombinator.com/showhn.html
- Product Hunt Best Practices: https://www.lennysnewsletter.com/p/how-to-successfully-launch-on-product
- Reddit Self-Promotion Guide: https://jetthoughts.com/blog/self-promote-on-reddit-without-getting-banned-promotion/
- Twitter Growth Guide: https://www.wisp.blog/blog/top-indie-hackers-to-follow-on-twitter-in-2024
- GitHub README Best Practices: https://blog.beautifulmarkdown.com/10-github-readme-examples-that-get-stars

**Tools:**
- VHS: https://github.com/charmbracelet/vhs
- Star History: https://star-history.com/
- Hotpot.ai PH Gallery: https://hotpot.ai/templates/product-hunt-gallery
- Typefully: https://typefully.com/
- OBS Studio: https://obsproject.com/

**Communities:**
- r/commandline: https://reddit.com/r/commandline
- r/ClaudeAI: https://reddit.com/r/ClaudeAI
- r/SideProject: https://reddit.com/r/SideProject
- r/golang: https://reddit.com/r/golang
- Charm Community: https://charm.sh/

**Analytics:**
- GitHub Traffic: Insights → Traffic
- Twitter Analytics: analytics.twitter.com
- YouTube Analytics: studio.youtube.com/analytics
- Google Analytics: analytics.google.com

---

*Last Updated: December 24, 2025*
*Version: 1.0*
*Author: Comprehensive market research across HN, PH, Twitter, Reddit, GitHub*
