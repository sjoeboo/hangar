# TUI Rendering Bug Analysis: Duplicated Footer

## Issue Description
The agent-deck TUI is displaying the footer/help bar **4 times** at the bottom of the screen, creating a stacked/duplicated appearance.

![Screenshot showing duplicated footer]
```
Session: Enter Attach n/N New/Quick g Group R Res
Group: Tab Toggle n/New Quick g Group r Renam
Session: Enter Attach n/N New/Quick g Group R Res
Session: Enter Attach n/N New/Quick g Group R Res
```

## Root Cause

### 1. Terminal Dimension Issue
The terminal is reporting **zero dimensions**:
```bash
$ echo "COLUMNS: $COLUMNS | LINES: $LINES"
COLUMNS: 0 | LINES: 0
```

This occurs when:
- Running inside tmux without proper terminal size initialization
- Terminal window is being resized
- SSH session with improperly forwarded terminal size
- After certain tmux operations (detach/reattach, pane splits)

### 2. Bubble Tea Rendering Bug
From `internal/ui/home.go:5662-5667`:
```go
// CRITICAL: Use MaxWidth (truncate) instead of Width (word-wrap).
// Width wraps long lines, which INCREASES the total line count beyond h.height.
// When View() returns more lines than the terminal height, Bubble Tea's renderer
// loses cursor position tracking, causing all subsequent frames to stack on top of
// each other — making the entire view appear duplicated 2-4x.
```

When `h.height = 0`:
1. `ensureExactHeight(content, 0)` returns empty string (line 5660)
2. But `MaxHeight(0)` constraint doesn't prevent rendering
3. The help bar still renders with partial content
4. Bubble Tea loses cursor tracking → frames stack → duplication

### 3. Code Flow
```
WindowSizeMsg received → h.width=0, h.height=0 (line 2066-2067)
                    ↓
View() called → Guard at line 5410 should return "Loading..."
                    ↓
BUT: Race condition or partial render allows helpBar to render
                    ↓
renderHelpBar() → routes to renderHelpBarTiny() (width < 50)
                    ↓
Renders border with strings.Repeat("─", 0) → empty border
                    ↓
ensureExactHeight(content, 0) → returns ""
                    ↓
MaxWidth(0).MaxHeight(0) → doesn't prevent output
                    ↓
Bubble Tea renderer sees output > height → STACKING BUG
```

## Why the Guard Failed

The guard at line 5410 checks `if h.width == 0`, but:
1. There may be a **race condition** where width is set after the check
2. Or width is set to a **very small non-zero value** (e.g., 1-5) that passes the guard but still causes issues
3. The `minTerminalWidth` check at line 5415 should catch this, but may not work if height is also 0

## Immediate Workarounds

### Option 1: Force Terminal Resize (Quickest)
```bash
# Inside tmux, press Ctrl+b then type:
:resize-pane -Z    # Toggle zoom (fixes dimension detection)

# Or resize the terminal window manually
```

### Option 2: Restart agent-deck
```bash
# Exit agent-deck (q)
# Restart
agent-deck
```

### Option 3: Set Terminal Size Explicitly
```bash
# Before starting agent-deck:
stty size          # Check current size
stty rows 40 cols 120  # Set explicit size if needed
agent-deck
```

## Proper Fixes

### Fix 1: Strengthen Dimension Guards
**File**: `internal/ui/home.go`
**Location**: Lines 5410-5427 (View function)

```go
func (h *Home) View() string {
	// CRITICAL: Return empty during attach
	if h.isAttaching.Load() {
		return ""
	}

	// ENHANCED: Guard against zero or invalid dimensions
	if h.width == 0 || h.height == 0 {
		// Don't try to render anything - return minimal loading message
		// This prevents the stacking bug from invalid dimensions
		return "Initializing terminal..."
	}

	// Check minimum terminal size for usability
	if h.width < minTerminalWidth || h.height < minTerminalHeight {
		// Create a simple error message without using lipgloss.Place
		// which might fail with zero dimensions
		msg := fmt.Sprintf("Terminal too small (%dx%d)\\nMinimum: %dx%d",
			h.width, h.height, minTerminalWidth, minTerminalHeight)

		// Only use lipgloss.Place if dimensions are sane
		if h.width > 0 && h.height > 0 {
			return lipgloss.Place(h.width, h.height,
				lipgloss.Center, lipgloss.Center,
				lipgloss.NewStyle().Foreground(ColorYellow).Render(msg))
		}
		return msg
	}

	// ... rest of View() ...
}
```

### Fix 2: Validate Dimensions in WindowSizeMsg Handler
**File**: `internal/ui/home.go`
**Location**: Lines 2065-2073 (Update function)

```go
case tea.WindowSizeMsg:
	// Validate dimensions before applying
	if msg.Width <= 0 || msg.Height <= 0 {
		// Invalid dimensions - don't update
		// Log for debugging
		uiLog.Warn("invalid_window_size",
			slog.Int("width", msg.Width),
			slog.Int("height", msg.Height))
		return h, nil
	}

	h.width = msg.Width
	h.height = msg.Height
	h.updateSizes()
	h.syncViewport()
	h.setupWizard.SetSize(msg.Width, msg.Height)
	h.settingsPanel.SetSize(msg.Width, msg.Height)
	h.geminiModelDialog.SetSize(msg.Width, msg.Height)
	return h, nil
```

### Fix 3: Add Dimension Validation in ensureExactHeight
**File**: `internal/ui/home.go`
**Location**: Lines 5983-6004

```go
func ensureExactHeight(content string, n int) string {
	// ENHANCED: Validate input
	if n <= 0 {
		// Return minimal content instead of empty string
		// This prevents the stacking bug
		return "Loading..."
	}

	// Split into lines
	lines := strings.Split(content, "\\n")

	// Truncate or pad to exactly n lines
	if len(lines) > n {
		lines = lines[:n]
	} else if len(lines) < n {
		for len(lines) < n {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\\n")
}
```

### Fix 4: Initialize Terminal Size on Startup
**File**: `internal/ui/home.go`
**Location**: In the initialization function (where Home is created)

```go
// In New() or Init():
func (h *Home) Init() tea.Cmd {
	// Force immediate terminal size detection
	// This ensures we never render with 0x0 dimensions
	return tea.Batch(
		tea.EnterAltScreen,
		tea.WindowSize(), // Request current terminal size
		h.tick(),
		h.loadSessions,
	)
}
```

## Testing the Fix

After applying fixes, test these scenarios:
1. Start agent-deck in a fresh tmux pane
2. Resize terminal window while agent-deck is running
3. Detach and reattach to tmux session
4. Split tmux panes while agent-deck is open
5. SSH to remote machine and run agent-deck
6. Start with very small terminal (20x10) and gradually resize

Expected: No footer duplication in any scenario.

## Related Code Comments

The codebase already has awareness of this issue:
- Line 5657-5672: Detailed comment about View() height constraint issues
- Line 5404-5408: Guard against output during attach
- Line 2065: WindowSizeMsg handler

The fixes build on this existing understanding and strengthen the guards.

## Detection Script

To check if this issue is affecting a system:

```bash
#!/bin/bash
# Check for zero terminal dimensions in tmux

if [ -n "$TMUX" ]; then
    echo "Running in tmux"
    width=$(tput cols 2>/dev/null || echo "0")
    height=$(tput lines 2>/dev/null || echo "0")
    echo "Dimensions: ${width}x${height}"

    if [ "$width" -eq 0 ] || [ "$height" -eq 0 ]; then
        echo "⚠️  WARNING: Zero dimensions detected!"
        echo "This will cause agent-deck TUI rendering issues."
        echo "Fix: Press Ctrl+b :resize-pane -Z"
    else
        echo "✓ Dimensions OK"
    fi
else
    echo "Not in tmux"
fi
```

## References
- Bubble Tea Issue #431: View() output leakage during tea.Exec
- agent-deck commit efe3391: "fix(claude): sync session ID after /clear, /fork, /compact"
- This issue appears related to tmux dimension handling, not compact command
