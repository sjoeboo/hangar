# PR Overview View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a full-screen PR overview mode (toggled with `P`) that lists only sessions with active PRs in a compact format with PR state and CI check status.

**Architecture:** Add `viewMode string` and `prViewCursor int` fields to the `Home` struct. `View()` routes to a new `renderPROverview()` function when `viewMode == "prs"`. Key routing is handled in a new `handlePRViewKey()` function called before `handleMainKey()`. All data comes from the existing `prCache` map — no new fetch infrastructure needed.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), lipgloss (`github.com/charmbracelet/lipgloss`), existing `prCacheEntry` type, `gh` CLI (already integrated)

---

### Task 1: Add state fields and helper to Home struct

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add the two new fields to the `Home` struct**

Find the `// State` comment block around line 151. Add after the `sortMode` field:

```go
// PR overview view
viewMode     string // "" or "sessions" = normal, "prs" = PR overview full-screen
prViewCursor int    // cursor position within PR overview list
```

**Step 2: Add a helper that returns the filtered PR list**

Add this new function near the other small helper functions (around line 339, after `getLayoutMode`):

```go
// prViewSessions returns sessions that have a non-nil PR cache entry, in flatItems order.
func (h *Home) prViewSessions() []*session.Instance {
	h.prCacheMu.Lock()
	defer h.prCacheMu.Unlock()
	var result []*session.Instance
	for _, item := range h.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil {
			if pr, ok := h.prCache[item.Session.ID]; ok && pr != nil {
				result = append(result, item.Session)
			}
		}
	}
	return result
}
```

**Step 3: Run the tests to make sure nothing broke**

```bash
go test ./internal/ui/... 2>&1 | grep -E "FAIL|ok|error"
```

Expected: same pass/fail profile as before (two pre-existing failures are OK).

**Step 4: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(pr-view): add viewMode/prViewCursor fields and prViewSessions helper"
```

---

### Task 2: Wire `P` key to toggle PR view in `handleMainKey`

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add the `P` case to `handleMainKey`**

Inside `handleMainKey` (around line 3759), add a new case before the `case "q", "ctrl+c":` line:

```go
case "P":
	if h.ghPath != "" {
		h.viewMode = "prs"
		h.prViewCursor = 0
		// Trigger fetches for any session missing recent PR data
		var cmds []tea.Cmd
		for _, item := range h.flatItems {
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.IsWorktree() && item.Session.WorktreePath != "" {
				h.prCacheMu.Lock()
				cacheTs, hasCached := h.prCacheTs[item.Session.ID]
				needsFetch := !hasCached || time.Since(cacheTs) > 60*time.Second
				if needsFetch {
					h.prCacheTs[item.Session.ID] = time.Now()
				}
				h.prCacheMu.Unlock()
				if needsFetch {
					sid := item.Session.ID
					wtPath := item.Session.WorktreePath
					ghPath := h.ghPath
					cmds = append(cmds, func() tea.Msg {
						return fetchPRInfo(sid, wtPath, ghPath)
					})
				}
			}
		}
		return h, tea.Batch(cmds...)
	}
	return h, nil
```

**Step 2: Write a test for the toggle**

In `internal/ui/home_test.go`, add:

```go
func TestPRView_ToggleWithP(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.ghPath = "/usr/bin/gh" // fake path — just needs to be non-empty

	if home.viewMode == "prs" {
		t.Fatal("viewMode should not start in prs mode")
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode != "prs" {
		t.Errorf("viewMode = %q, want \"prs\"", h.viewMode)
	}
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor = %d, want 0", h.prViewCursor)
	}
}

func TestPRView_NoToggleWithoutGh(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	// ghPath is "" by default

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode == "prs" {
		t.Error("viewMode should not switch to prs when gh is not installed")
	}
}
```

**Step 3: Run the new tests**

```bash
go test ./internal/ui/... -run TestPRView_Toggle -v
```

Expected: both tests PASS.

**Step 4: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(pr-view): wire P key to enter PR overview mode"
```

---

### Task 3: Add `handlePRViewKey` function

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add the key handler function**

Add this new function after `handleMainKey`. It handles all keys while `viewMode == "prs"`:

```go
// handlePRViewKey handles keys when the PR overview view is active.
func (h *Home) handlePRViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sessions := h.prViewSessions()

	switch msg.String() {
	case "P", "esc":
		h.viewMode = ""
		return h, nil

	case "up", "k":
		if h.prViewCursor > 0 {
			h.prViewCursor--
		}
		return h, nil

	case "down", "j":
		if h.prViewCursor < len(sessions)-1 {
			h.prViewCursor++
		}
		return h, nil

	case "enter":
		if h.prViewCursor < len(sessions) {
			inst := sessions[h.prViewCursor]
			if inst.Exists() {
				h.isAttaching.Store(true)
				return h, h.attachSession(inst)
			}
		}
		return h, nil

	case "o":
		if h.prViewCursor < len(sessions) {
			inst := sessions[h.prViewCursor]
			h.prCacheMu.Lock()
			pr, hasPR := h.prCache[inst.ID]
			h.prCacheMu.Unlock()
			if hasPR && pr != nil && pr.URL != "" {
				exec.Command("open", pr.URL).Start() //nolint:errcheck
			}
		}
		return h, nil

	case "r":
		// Force re-fetch PR data for all sessions
		var cmds []tea.Cmd
		for _, inst := range sessions {
			if inst.IsWorktree() && inst.WorktreePath != "" {
				h.prCacheMu.Lock()
				h.prCacheTs[inst.ID] = time.Time{} // reset TTL
				h.prCacheMu.Unlock()
				sid := inst.ID
				wtPath := inst.WorktreePath
				ghPath := h.ghPath
				cmds = append(cmds, func() tea.Msg {
					return fetchPRInfo(sid, wtPath, ghPath)
				})
			}
		}
		return h, tea.Batch(cmds...)
	}

	return h, nil
}
```

**Step 2: Wire `handlePRViewKey` into the main `Update` dispatch**

Find the `handleMainKey` call in the `Update` function (around line 3344):

```go
// Main view keys
return h.handleMainKey(msg)
```

Change it to:

```go
// PR overview view
if h.viewMode == "prs" {
	return h.handlePRViewKey(msg)
}
// Main view keys
return h.handleMainKey(msg)
```

**Step 3: Write tests for PR view key navigation**

In `internal/ui/home_test.go`, add:

```go
func TestPRView_EscReturnsToSessions(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.viewMode = "prs"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := home.Update(msg)
	h := model.(*Home)

	if h.viewMode == "prs" {
		t.Error("esc should exit PR view")
	}
}

func TestPRView_Navigation(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 40
	home.viewMode = "prs"

	// Seed PR cache with two sessions
	sess1 := &session.Instance{ID: "s1", Title: "Session 1", WorktreePath: "/tmp/s1"}
	sess2 := &session.Instance{ID: "s2", Title: "Session 2", WorktreePath: "/tmp/s2"}
	home.instances = []*session.Instance{sess1, sess2}
	home.rebuildFlatItems()
	home.prCache["s1"] = &prCacheEntry{Number: 1, Title: "PR 1", State: "OPEN", URL: "https://github.com/x/y/pull/1"}
	home.prCache["s2"] = &prCacheEntry{Number: 2, Title: "PR 2", State: "DRAFT", URL: "https://github.com/x/y/pull/2"}

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := home.Update(msg)
	h := model.(*Home)
	if h.prViewCursor != 1 {
		t.Errorf("prViewCursor after down = %d, want 1", h.prViewCursor)
	}

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = h.Update(msg)
	h = model.(*Home)
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor after up = %d, want 0", h.prViewCursor)
	}

	// Navigate up at top — stays at 0
	model, _ = h.Update(msg)
	h = model.(*Home)
	if h.prViewCursor != 0 {
		t.Errorf("prViewCursor should clamp at 0, got %d", h.prViewCursor)
	}
}
```

**Step 4: Run the new tests**

```bash
go test ./internal/ui/... -run "TestPRView_" -v
```

Expected: all four `TestPRView_` tests PASS.

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(pr-view): add handlePRViewKey with nav/attach/open/refresh"
```

---

### Task 4: Implement `renderPROverview`

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add the render function**

Add `renderPROverview` after `renderHelpBarFull` (around line 6989). It renders the full-screen PR list:

```go
// renderPROverview renders the full-screen PR overview mode.
func (h *Home) renderPROverview() string {
	sessions := h.prViewSessions()

	var b strings.Builder

	// ── Header ──────────────────────────────────────────────────────────
	running, waiting, idle, _ := h.countSessionStatuses()
	logo := RenderLogoCompact(running, waiting, idle)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	viewLabel := lipgloss.NewStyle().Foreground(ColorPurple).Bold(true).Render("PR Overview")
	headerRight := lipgloss.NewStyle().Foreground(ColorComment).Render(fmt.Sprintf("%d PRs", len(sessions)))
	headerLeft := logo + "  " + titleStyle.Render("Hangar") + "  " + viewLabel
	pad := h.width - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if pad < 1 {
		pad = 1
	}
	b.WriteString(headerLeft + strings.Repeat(" ", pad) + headerRight)
	b.WriteString("\n")

	// ── Column header ────────────────────────────────────────────────────
	colStyle := lipgloss.NewStyle().Foreground(ColorComment).Bold(true)
	b.WriteString(colStyle.Render(fmt.Sprintf("  %-7s  %-8s  %-14s  %s", "PR", "STATE", "CHECKS", "SESSION")))
	b.WriteString("\n")
	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	b.WriteString(borderStyle.Render(strings.Repeat("─", max(0, h.width))))
	b.WriteString("\n")

	// ── Rows ─────────────────────────────────────────────────────────────
	contentHeight := h.height - 5 // header(1) + colheader(1) + border(1) + helpbar(2)
	startIdx := 0
	if h.prViewCursor >= contentHeight {
		startIdx = h.prViewCursor - contentHeight + 1
	}

	if len(sessions) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(h.width, lipgloss.Center, emptyStyle.Render("No sessions with open PRs")))
		b.WriteString("\n")
	}

	for i := startIdx; i < len(sessions) && i < startIdx+contentHeight; i++ {
		inst := sessions[i]
		selected := i == h.prViewCursor

		h.prCacheMu.Lock()
		pr := h.prCache[inst.ID]
		h.prCacheMu.Unlock()

		// PR number
		prNumStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		prNum := prNumStyle.Render(fmt.Sprintf("#%-5d", pr.Number))

		// State
		var stateStyle lipgloss.Style
		var stateLabel string
		switch pr.State {
		case "OPEN":
			stateStyle = lipgloss.NewStyle().Foreground(ColorGreen)
			stateLabel = "open"
		case "DRAFT":
			stateStyle = lipgloss.NewStyle().Foreground(ColorComment)
			stateLabel = "draft"
		case "MERGED":
			stateStyle = lipgloss.NewStyle().Foreground(ColorPurple)
			stateLabel = "merged"
		case "CLOSED":
			stateStyle = lipgloss.NewStyle().Foreground(ColorRed)
			stateLabel = "closed"
		default:
			stateStyle = lipgloss.NewStyle().Foreground(ColorComment)
			stateLabel = strings.ToLower(pr.State)
		}
		stateCol := stateStyle.Render(fmt.Sprintf("%-8s", stateLabel))

		// Checks
		var checkParts []string
		if pr.HasChecks {
			if pr.ChecksFailed > 0 {
				checkParts = append(checkParts, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✗%d", pr.ChecksFailed)))
			}
			if pr.ChecksPending > 0 {
				checkParts = append(checkParts, lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("●%d", pr.ChecksPending)))
			}
			if pr.ChecksPassed > 0 {
				checkParts = append(checkParts, lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("✓%d", pr.ChecksPassed)))
			}
		}
		checksRaw := strings.Join(checkParts, " ")
		// Pad checks column to fixed width (14 visible chars)
		checksVisible := lipgloss.Width(checksRaw)
		if checksVisible < 14 {
			checksRaw += strings.Repeat(" ", 14-checksVisible)
		}

		// Session title — bright if running, dim if idle
		status := inst.GetStatusThreadSafe()
		var titleStyle lipgloss.Style
		if status == session.StatusRunning {
			titleStyle = lipgloss.NewStyle().Foreground(ColorForeground).Bold(true)
		} else {
			titleStyle = lipgloss.NewStyle().Foreground(ColorComment)
		}
		title := inst.Title
		maxTitleWidth := h.width - 2 - 7 - 2 - 8 - 2 - 14 - 2 - 4
		if maxTitleWidth > 0 && len(title) > maxTitleWidth {
			title = title[:maxTitleWidth-1] + "…"
		}
		sessionCol := titleStyle.Render(title)

		row := fmt.Sprintf("  %s  %s  %s  %s", prNum, stateCol, checksRaw, sessionCol)

		if selected {
			row = lipgloss.NewStyle().
				Background(ColorSelection).
				Width(h.width).
				Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	// Fill remaining lines
	rendered := strings.Count(b.String(), "\n") - 3 // subtract header lines
	for i := rendered; i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// ── Help bar ─────────────────────────────────────────────────────────
	borderLine := borderStyle.Render(strings.Repeat("─", max(0, h.width)))
	b.WriteString(borderLine)
	b.WriteString("\n")
	helpKeyStyle := lipgloss.NewStyle().Foreground(ColorForeground).Bold(true)
	helpDescStyle := lipgloss.NewStyle().Foreground(ColorComment)
	renderKey := func(key, desc string) string {
		return helpKeyStyle.Render(key) + " " + helpDescStyle.Render(desc)
	}
	sepStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	sep := sepStyle.Render(" │ ")
	hints := strings.Join([]string{
		renderKey("Enter", "Attach"),
		renderKey("o", "Open PR"),
		renderKey("r", "Refresh"),
		renderKey("P/Esc", "Back"),
		renderKey("↑↓/jk", "Nav"),
	}, sep)
	b.WriteString(hints)

	return lipgloss.NewStyle().MaxWidth(h.width).Render(b.String())
}
```

**Step 2: Wire `renderPROverview` into `View()`**

In `View()`, after the `if h.todoDialog.IsVisible()` block (around line 5847), add:

```go
// PR overview takes full screen
if h.viewMode == "prs" {
	return h.renderPROverview()
}
```

**Step 3: Write a render test**

In `internal/ui/home_test.go`, add:

```go
func TestPRView_RenderShowsPRs(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.viewMode = "prs"
	home.ghPath = "/usr/bin/gh"

	sess1 := &session.Instance{ID: "s1", Title: "Fix auth bug", WorktreePath: "/tmp/s1"}
	home.instances = []*session.Instance{sess1}
	home.rebuildFlatItems()
	home.prCache["s1"] = &prCacheEntry{
		Number:       42,
		Title:        "Fix auth bug",
		State:        "OPEN",
		URL:          "https://github.com/x/y/pull/42",
		HasChecks:    true,
		ChecksPassed: 5,
		ChecksFailed: 1,
	}

	view := home.View()

	if !strings.Contains(view, "#42") {
		t.Error("View should contain PR number #42")
	}
	if !strings.Contains(view, "open") {
		t.Error("View should contain PR state 'open'")
	}
	if !strings.Contains(view, "Fix auth bug") {
		t.Error("View should contain session title")
	}
	if !strings.Contains(view, "PR Overview") {
		t.Error("View should show 'PR Overview' header label")
	}
}

func TestPRView_RenderEmpty(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.viewMode = "prs"

	view := home.View()

	if !strings.Contains(view, "No sessions") {
		t.Error("Empty PR view should show empty state message")
	}
}
```

**Step 4: Run all PR view tests**

```bash
go test ./internal/ui/... -run "TestPRView_" -v
```

Expected: all `TestPRView_` tests PASS.

**Step 5: Build and smoke test**

```bash
go build ./...
```

Expected: no compile errors.

**Step 6: Commit**

```bash
git add internal/ui/home.go internal/ui/home_test.go
git commit -m "feat(pr-view): implement renderPROverview and wire into View()"
```

---

### Task 5: Add `P` hint to main view help bar

**Files:**
- Modify: `internal/ui/home.go`

**Step 1: Add `P` hint in `renderHelpBarFull`**

In `renderHelpBarFull` (around line 6968), the global hints line reads:

```go
globalHints := globalStyle.Render("↑↓ Nav") + sep +
    globalStyle.Render("/ Search  G Global") + sep +
    globalStyle.Render("? Help  q Quit")
```

Change it to include the PR view hint when `gh` is available:

```go
searchHint := "/ Search  G Global"
if h.ghPath != "" {
    searchHint += "  P PRs"
}
globalHints := globalStyle.Render("↑↓ Nav") + sep +
    globalStyle.Render(searchHint) + sep +
    globalStyle.Render("? Help  q Quit")
```

**Step 2: Add `P` hint in `renderHelpBarCompact` (medium terminals)**

Find `renderHelpBarCompact` and locate its global hints section. Apply the same pattern — add `P PRs` to the global hints string when `h.ghPath != ""`.

**Step 3: Run full test suite**

```bash
go test ./internal/ui/... 2>&1 | grep -E "FAIL|ok|---"
```

Expected: same pass/fail profile as baseline (two pre-existing failures OK).

**Step 4: Build final check**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(pr-view): add P hint to help bar when gh is available"
```

---

### Task 6: Manual smoke test

**Step 1: Run Hangar locally**

```bash
go run ./cmd/hangar
```

**Step 2: Verify the following**

- [ ] `P` key switches to PR overview (only if `gh` is in PATH)
- [ ] Only sessions with PRs appear; sessions without PRs are hidden
- [ ] Each row shows: `#NNN  state  checks  title`
- [ ] `↑`/`↓`/`j`/`k` navigates the list
- [ ] `enter` attaches to the selected session
- [ ] `o` opens the PR URL in the browser
- [ ] `r` triggers a refresh (rows may flicker as data reloads)
- [ ] `P` or `esc` returns to the normal sessions view
- [ ] Help bar in normal view shows `P PRs` hint when `gh` is installed
- [ ] `P` in normal view does nothing if `gh` is not installed

**Step 3: If all checks pass, the feature is complete.**
