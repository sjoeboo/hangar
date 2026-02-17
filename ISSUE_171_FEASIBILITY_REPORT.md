# GitHub Issue #171: Show All Sessions in Notification Bar
## Feasibility Analysis & Implementation Report

**Issue URL**: https://github.com/asheshgoplani/agent-deck/issues/171
**Author**: msohaildev
**Status**: OPEN
**Date Analyzed**: 2026-02-16

---

## Executive Summary

✅ **Recommendation**: **IMPLEMENT** - This is a well-scoped, low-risk enhancement with clear user value.

- **Feasibility**: HIGH - The architecture supports this change cleanly
- **Effort**: SMALL (~2-3 hours) - 4 files, ~50-60 lines of code changes
- **Risk**: LOW - Opt-in feature with no breaking changes
- **User Value**: HIGH - Significantly improves multi-session workflow visibility

---

## What the Issue Asks For

### Current Behavior
The tmux notification bar **only shows sessions in "waiting" state**:
```
⚡ [1] test               # Only when status = StatusWaiting
```

When you attach to that session, the bar clears:
```
                          # Empty - no visibility into other sessions
```

### Desired Behavior
Add a config option `show_all` under `[notifications]` that displays **all sessions** (excluding the currently attached one) with status indicators:

```toml
[notifications]
  enabled = true
  show_all = true    # New option (default: false for backward compatibility)
```

Bar output with `show_all = true`:
```
⚡ [1] ● frontend [2] ◐ api [3] ○ backend
```

**Status Icons**:
- `●` = running (green)
- `◐` = waiting (yellow)
- `○` = idle (gray)
- `✕` = error (red)

### Use Case
When running multiple concurrent AI coding sessions, users want to see **all active sessions** in the tmux status bar for instant navigation via `Ctrl+b 1-6`, not just the ones currently waiting for input.

---

## Current Architecture

### 1. Configuration Layer (`internal/session/userconfig.go`)

**Current Structure** (lines 257-264):
```go
type NotificationsConfig struct {
    Enabled bool `toml:"enabled"`     // Show notification bar (default: true)
    MaxShown int `toml:"max_shown"`   // Max sessions shown (default: 6)
}
```

**What it controls**:
- Whether notifications are enabled
- Maximum number of sessions to show (currently 6)

### 2. Notification Manager (`internal/session/notifications.go`)

**Key Components**:

**NotificationEntry** (lines 11-18):
```go
type NotificationEntry struct {
    SessionID    string
    TmuxName     string
    Title        string
    AssignedKey  string        // "1", "2", "3", etc.
    WaitingSince time.Time
}
```

**NotificationManager** (lines 20-36):
- Tracks waiting sessions
- Manages key assignments (1-6)
- Formats the status bar text

**Critical Filter Logic** - `SyncFromInstances()` (line 176):
```go
// Current: Only add waiting sessions
if inst.GetStatusThreadSafe() == StatusWaiting && inst.ID != currentSessionID {
    waitingSet[inst.ID] = inst
}
```

**Bar Formatting** - `FormatBar()` (lines 149-165):
```go
func (nm *NotificationManager) FormatBar() string {
    // Returns: "⚡ [1] frontend [2] api"
    for _, e := range nm.entries {
        parts = append(parts, fmt.Sprintf("[%s] %s", e.AssignedKey, e.Title))
    }
    return "⚡ " + strings.Join(parts, " ")
}
```

**Note**: Currently no status icons in the bar format - just key and title.

### 3. UI Integration (`internal/ui/home.go`)

**Notification Manager Initialization** (line 556):
```go
h.notificationManager = session.NewNotificationManager(notifSettings.MaxShown)
```

**Periodic Sync** (line 1848):
```go
h.notificationManager.SyncFromInstances(instances, currentSessionID)
barText := h.notificationManager.FormatBar()
// Updates tmux status-left with the formatted bar
```

**Status Icon Rendering** (lines 6835-6855):
```go
// Already exists for list view - can reuse!
switch instStatus {
case session.StatusRunning:
    statusIcon = "●"
    statusStyle = SessionStatusRunning
case session.StatusWaiting:
    statusIcon = "◐"
    statusStyle = SessionStatusWaiting
case session.StatusIdle:
    statusIcon = "○"
    statusStyle = SessionStatusIdle
case session.StatusError:
    statusIcon = "✕"
    statusStyle = SessionStatusError
}
```

### 4. Session Status Types (`internal/session/instance.go`)

**Available Statuses** (lines 32-35):
```go
StatusRunning  Status = "running"   // ● Green
StatusWaiting  Status = "waiting"   // ◐ Yellow
StatusIdle     Status = "idle"      // ○ Gray
StatusError    Status = "error"     // ✕ Red
```

---

## Implementation Plan

### Files to Modify

1. **`internal/session/userconfig.go`** (~5 lines added)
2. **`internal/session/notifications.go`** (~25-30 lines modified)
3. **`internal/ui/home.go`** (~5-10 lines modified)
4. **Tests** (optional): `internal/session/notifications_test.go` (~20-30 lines)

### Total Estimated Changes
- **Core Implementation**: ~40-50 lines of code
- **With Tests**: ~70-80 lines of code
- **Effort**: 2-3 hours (including testing)

---

## Detailed Changes Required

### 1. Config Schema (`internal/session/userconfig.go`)

**Add `ShowAll` field to `NotificationsConfig`** (line 264):

```go
type NotificationsConfig struct {
    Enabled bool `toml:"enabled"`
    MaxShown int `toml:"max_shown"`
    ShowAll bool `toml:"show_all"`  // NEW: Show all sessions with status icons
}
```

**Update config getter** (around line 1161):
```go
func GetNotificationsSettings() NotificationsConfig {
    // ... existing code ...
    // ShowAll defaults to false (backward compatible)
    if !settings.Enabled && settings.MaxShown == 0 && !settings.ShowAll {
        settings.Enabled = true
    }
    return settings
}
```

**Update example config** (line 1511):
```go
# [notifications]
# enabled = true
# max_shown = 6
# show_all = true  # Show all sessions with status icons, not just waiting
```

### 2. Notification Manager (`internal/session/notifications.go`)

**A. Add `showAll` field to struct** (around line 24):

```go
type NotificationManager struct {
    entries  []*NotificationEntry
    maxShown int
    showAll  bool  // NEW: Show all sessions vs only waiting
    mu       sync.RWMutex
}
```

**B. Update constructor** (line 28):

```go
func NewNotificationManager(maxShown int, showAll bool) *NotificationManager {
    if maxShown <= 0 {
        maxShown = 6
    }
    return &NotificationManager{
        entries:  make([]*NotificationEntry, 0),
        maxShown: maxShown,
        showAll:  showAll,  // NEW
    }
}
```

**C. Add status field to `NotificationEntry`** (around line 17):

```go
type NotificationEntry struct {
    SessionID    string
    TmuxName     string
    Title        string
    AssignedKey  string
    WaitingSince time.Time
    Status       Status  // NEW: Track status for icon rendering
}
```

**D. Update filter logic in `SyncFromInstances()`** (line 176):

```go
// Build set of sessions to show (excluding current)
sessionSet := make(map[string]*Instance)
for _, inst := range instances {
    // Skip current session
    if inst.ID == currentSessionID {
        continue
    }

    // Filter based on mode
    if nm.showAll {
        // Show all sessions regardless of status
        sessionSet[inst.ID] = inst
    } else {
        // Original behavior: only waiting sessions
        if inst.GetStatusThreadSafe() == StatusWaiting {
            sessionSet[inst.ID] = inst
        }
    }
}
```

**E. Update entry creation to include status** (around line 199):

```go
entry := &NotificationEntry{
    SessionID:    inst.ID,
    TmuxName:     tmuxName,
    Title:        inst.Title,
    WaitingSince: inst.GetWaitingSince(),
    Status:       inst.GetStatusThreadSafe(),  // NEW
}
```

**F. Update `FormatBar()` to include status icons** (line 150):

```go
func (nm *NotificationManager) FormatBar() string {
    nm.mu.RLock()
    defer nm.mu.RUnlock()

    if len(nm.entries) == 0 {
        return ""
    }

    var parts []string
    for _, e := range nm.entries {
        var statusIcon string
        if nm.showAll {
            // Include status icon when showing all sessions
            switch e.Status {
            case StatusRunning:
                statusIcon = "● "
            case StatusWaiting:
                statusIcon = "◐ "
            case StatusIdle:
                statusIcon = "○ "
            case StatusError:
                statusIcon = "✕ "
            default:
                statusIcon = "○ "
            }
            parts = append(parts, fmt.Sprintf("[%s] %s%s", e.AssignedKey, statusIcon, e.Title))
        } else {
            // Original format: no status icon (all are waiting)
            parts = append(parts, fmt.Sprintf("[%s] %s", e.AssignedKey, e.Title))
        }
    }

    return "⚡ " + strings.Join(parts, " ")
}
```

### 3. UI Layer (`internal/ui/home.go`)

**Update initialization** (line 556):

```go
h.notificationManager = session.NewNotificationManager(
    notifSettings.MaxShown,
    notifSettings.ShowAll,  // NEW: Pass show_all config
)
```

**No other changes needed** - the sync and update logic remains the same.

---

## Benefits & User Value

### Immediate Benefits
1. **Better Visibility**: See all active sessions at a glance
2. **Faster Navigation**: Jump to any session instantly via `Ctrl+b 1-6`
3. **Status Awareness**: Know which sessions are running, waiting, or idle
4. **Multi-Session Workflow**: Essential for users juggling 3+ AI sessions

### Workflow Example

**Before** (current):
```
# Working in session A - bar is empty
$ # No idea what session B or C are doing
$ # Have to detach and check TUI to see other sessions
```

**After** (with show_all):
```
# Working in session A
⚡ [1] ● session-B [2] ○ session-C [3] ◐ session-D
$ # Can see B is running, C is idle, D is waiting
$ # Press Ctrl+b 2 to jump to idle session C
```

---

## Risk Analysis

### Risks Identified
1. **Longer Status Bar**: More sessions = longer bar text
   - **Mitigation**: `max_shown` already limits to 6 sessions
   - **Mitigation**: tmux handles overflow with truncation

2. **Performance**: Syncing all sessions vs only waiting
   - **Impact**: Minimal - status is already tracked for all sessions
   - **Current**: `SyncFromInstances()` already iterates all instances

3. **Backward Compatibility**: Changing default behavior
   - **Mitigation**: Feature is opt-in via `show_all = false` default
   - **Mitigation**: Existing users see no change unless they enable it

### Breaking Changes
**NONE** - This is a pure additive feature with opt-in semantics.

---

## Testing Strategy

### Unit Tests (`internal/session/notifications_test.go`)

```go
func TestNotificationManager_ShowAll(t *testing.T) {
    // Test 1: show_all = false (original behavior)
    nm := NewNotificationManager(6, false)
    instances := []*Instance{
        createTestInstance("1", StatusRunning),
        createTestInstance("2", StatusWaiting),
        createTestInstance("3", StatusIdle),
    }
    nm.SyncFromInstances(instances, "")
    // Assert: Only "2" (waiting) is in entries

    // Test 2: show_all = true (new behavior)
    nm = NewNotificationManager(6, true)
    nm.SyncFromInstances(instances, "")
    // Assert: All 3 sessions are in entries

    // Test 3: Current session exclusion
    nm.SyncFromInstances(instances, "1")
    // Assert: "1" is excluded, "2" and "3" are included

    // Test 4: Status icon rendering
    bar := nm.FormatBar()
    // Assert: Contains "●", "◐", "○" when show_all = true
}
```

### Manual Testing
1. **Default behavior** (show_all = false):
   - Create 3 sessions
   - Only waiting sessions appear in bar
   - Bar clears when attached to waiting session

2. **Show all mode** (show_all = true):
   - Create 3 sessions (running, waiting, idle)
   - All sessions appear with status icons
   - Current session excluded from bar
   - Key bindings work (`Ctrl+b 1-6`)

3. **Edge cases**:
   - 0 sessions → empty bar
   - 10 sessions → only first 6 shown (max_shown)
   - All sessions same status → correct icons
   - Session transitions → bar updates correctly

---

## Alternative Approaches Considered

### Option 1: Separate Config Flag (Chosen)
**Approach**: Add `show_all` boolean to `[notifications]`
**Pros**: Simple, clear semantics, backward compatible
**Cons**: None identified
**Status**: ✅ Recommended

### Option 2: Mode Enum
**Approach**: `mode = "waiting_only" | "all_sessions"`
**Pros**: Extensible for future modes
**Cons**: Overkill for binary choice, more verbose config
**Status**: ❌ Over-engineered

### Option 3: Smart Auto-Detection
**Approach**: Automatically show all when >3 non-waiting sessions exist
**Pros**: No config needed
**Cons**: Unpredictable behavior, user has no control
**Status**: ❌ Too magical

---

## Implementation Checklist

- [ ] 1. Add `ShowAll bool` to `NotificationsConfig` struct
- [ ] 2. Update `GetNotificationsSettings()` with default
- [ ] 3. Add `showAll` field to `NotificationManager`
- [ ] 4. Update `NewNotificationManager()` constructor signature
- [ ] 5. Add `Status` field to `NotificationEntry`
- [ ] 6. Modify `SyncFromInstances()` filter logic
- [ ] 7. Update entry creation to include status
- [ ] 8. Modify `FormatBar()` to render status icons conditionally
- [ ] 9. Update `home.go` initialization to pass `show_all`
- [ ] 10. Add unit tests for new behavior
- [ ] 11. Update example config with new option
- [ ] 12. Manual testing (default + show_all modes)
- [ ] 13. Update documentation/changelog

---

## Related Issues & Features

### Similar Patterns in Codebase
- **Preview Mode Toggle**: `preview.show_output` and `preview.show_analytics`
  - Shows opt-in boolean config pattern works well
  - Located in same `userconfig.go` file

- **Status Filtering**: Existing status filter in TUI (StatusFilter field)
  - Shows status-based filtering is well-established
  - Could be extended if needed

### Future Enhancements (Out of Scope)
These are NOT part of this issue but could be added later:
1. **Custom Status Colors**: Allow users to customize icon colors
2. **Icon Customization**: Allow custom icons per status
3. **Bar Format Template**: Configurable bar format string
4. **Show/Hide Specific Statuses**: Fine-grained filter (e.g., hide idle)

---

## Timeline Estimate

| Phase | Time | Description |
|-------|------|-------------|
| **Implementation** | 1-1.5 hrs | Core code changes (4 files) |
| **Testing** | 0.5-1 hr | Unit tests + manual testing |
| **Documentation** | 0.5 hr | Update example config, comments |
| **Review & Polish** | 0.5 hr | Code review prep, edge cases |
| **TOTAL** | **2.5-3 hrs** | End-to-end implementation |

---

## Recommendation: GO

### Why This Should Be Implemented

✅ **High User Value**: Addresses a real pain point in multi-session workflows
✅ **Low Risk**: Opt-in feature with no breaking changes
✅ **Small Effort**: ~50 lines of code, well-scoped
✅ **Clean Design**: Fits naturally into existing architecture
✅ **Well-Specified**: Issue author provided clear implementation notes
✅ **Testable**: Easy to write comprehensive tests
✅ **Maintainable**: No complex logic or future debt

### Implementation Priority
**Priority**: MEDIUM-HIGH
**Rationale**: Small effort, high value, low risk = good ROI

### Success Criteria
1. Users can enable `show_all = true` in config
2. All sessions (except current) appear in bar with status icons
3. Original behavior (`show_all = false`) unchanged
4. Key bindings work correctly for all sessions
5. Bar updates in real-time as session statuses change
6. No performance degradation

---

## Appendix: Code Snippets

### Example User Config (After Implementation)

```toml
# ~/.agent-deck/config.toml

[notifications]
  enabled = true
  max_shown = 6
  show_all = true  # NEW: Show all sessions with status icons
```

### Expected Bar Output Examples

**Original mode** (`show_all = false`):
```
⚡ [1] frontend-feature [2] api-bugfix
```

**Show all mode** (`show_all = true`):
```
⚡ [1] ● frontend [2] ◐ api [3] ○ backend [4] ● ml-model [5] ○ docs
```

### Status Icon Legend
```
● Running  (ColorGreen)   - Session is actively working
◐ Waiting  (ColorYellow)  - Session is waiting for user input
○ Idle     (ColorGray)    - Session is idle/paused
✕ Error    (ColorRed)     - Session encountered an error
```

---

## Conclusion

This is a **well-designed, low-risk feature request** that significantly improves the multi-session workflow experience. The implementation is straightforward, the scope is clear, and the user value is high.

**Recommended Action**: Implement in the next sprint/development cycle.

---

**Report Generated**: 2026-02-16
**Analyzed By**: Claude (Sonnet 4.5)
**Review Status**: Ready for implementation
