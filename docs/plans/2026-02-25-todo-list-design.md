# Todo List Feature Design

**Date:** 2026-02-25
**Status:** Approved

## Overview

Add a per-project todo list to Hangar. Todos track work items tied to a project, with statuses that automatically transition based on session/worktree/PR lifecycle events. Users can create a session and worktree directly from a todo, and todos are cleaned up when their associated worktree is removed.

## Requirements

- Todos are scoped to a project (git repo)
- Statuses: `todo`, `in_progress`, `in_review`, `done`, `orphaned`
- Todos have a title and a description
- Create a session + worktree directly from a todo (branch named from todo title)
- 1:1 relationship between a todo and a session
- Auto-status transitions based on worktree/PR lifecycle
- Manual status override always available
- Todos are deleted when their worktree is removed via the finish flow
- If a session is manually deleted (not via worktree finish), todo becomes `orphaned`
- `t` key opens the todos view for the current project from anywhere in the session list

## Data Model

New `todos` table added to `~/.hangar/profiles/{profile}/state.db`:

```sql
CREATE TABLE todos (
    id           TEXT PRIMARY KEY,          -- 12-char hex, same pattern as instances
    project_path TEXT NOT NULL,             -- repo root path (matches WorktreeRepoRoot or ProjectPath)
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'todo',  -- todo | in_progress | in_review | done | orphaned
    session_id   TEXT NOT NULL DEFAULT '',  -- soft FK to instances.id (empty = unlinked)
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);
```

### Key decisions

- `project_path` is the repo root — for worktree sessions this is `WorktreeRepoRoot`, for normal sessions it is `ProjectPath`
- `session_id` is a soft reference (empty string = unlinked), consistent with existing Hangar patterns
- Todos are per-profile, which is correct since they link to sessions that are also per-profile
- Status is a plain string enum for DB readability

## Status Transitions

```
[todo] ──(create session+worktree)──→ [in_progress]
           ↑                                │
           │ (session manually deleted)     │ (open PR detected)
           │                                ↓
       [orphaned] ←──────────────── [in_review]
                                            │
                                            │ (PR merged / worktree finish)
                                            ↓
                                         [done]
                                            │
                              (worktree removed/cleaned up)
                                            ↓
                                       (deleted)
```

### Transition triggers

| Transition | Trigger |
|---|---|
| `todo → in_progress` | User creates session+worktree from the todo |
| `in_progress → in_review` | Open PR detected for linked session (existing PR status tick) |
| `in_review → done` | PR merged, detected on same tick |
| `any → orphaned` | Linked session is manually deleted (not via worktree finish) |
| `done → deleted` | Worktree removed via `worktree finish` or TUI cleanup flow |

All auto-transitions are overridable by the user via manual status selection (`s` key).

## TUI Integration

### Navigation

- Press `t` from any session row or group header → opens todos view for that project
- `esc` or `t` again → returns to session list
- The todos view replaces the session list (same pane, no split)

### Todos view layout

```
 Oliver                           [3 todos]
 ─────────────────────────────────────────────────────────────
 ● fix auth token refresh                          in progress
   └─ session: oliver-fix-auth-token-refresh ↗
 ○ add rate limiting to search API                      todo
 ✓ refactor user preferences storage                    done
 ! update onboarding flow                           orphaned
 ─────────────────────────────────────────────────────────────
 n new  enter open  s status  e edit  d delete  ? help
```

### Status icons (oasis_lagoon_dark palette)

| Icon | Status |
|---|---|
| `○` | todo — muted |
| `●` | in_progress — accent |
| `⟳` | in_review — amber |
| `✓` | done — green |
| `!` | orphaned — red/warning |

### Key bindings

| Key | Action |
|---|---|
| `n` | New todo (title + description prompt) |
| `enter` | If `in_progress`/`in_review`: attach to linked session. If `todo`: prompt to create session+worktree |
| `s` | Manually pick status |
| `e` | Edit title/description |
| `d` | Delete todo (with confirmation) |
| `↑` / `↓` | Navigate todos |
| `esc` / `t` | Back to session list |

## Creating a Session from a Todo

When the user presses `enter` on a `todo`-status item:

1. Creation dialog appears (reusing existing new-session dialog), pre-filled:
   - **Branch name**: slugified todo title (`"Fix auth token refresh"` → `fix-auth-token-refresh`)
   - **Project**: scoped to current project
   - **Worktree location**: project's default worktree strategy
2. User confirms (or edits branch name) → Hangar:
   - Creates the git worktree
   - Creates a new session with worktree fields set
   - Updates todo: `session_id = new session ID`, `status = in_progress`
   - Attaches to the new session
3. If the branch name already exists, inline error with editable field (same as existing worktree dialog UX)

## Implementation Notes

### Storage
- New migration in `internal/statedb/statedb.go` adds `todos` table
- New methods: `LoadTodos(projectPath)`, `SaveTodo(todo)`, `DeleteTodo(id)`, `UpdateTodoStatus(id, status)`

### PR/merge detection
- Piggyback on the existing per-session PR status tick
- When a session linked to a todo changes PR status: open → `in_review`, merged → `done`

### Worktree finish hook
- `cmd/hangar/worktree_cmd.go` finish flow: after worktree removal, find todo with matching `session_id` and delete it

### Orphan detection
- On session delete in `home.go`: find any todo with matching `session_id` → transition to `orphaned`

### New files
- `internal/session/todo.go` — `Todo` struct and status constants
- TUI todos view as a new mode in `home.go` (following existing dialog pattern)

### Tests
- Unit tests for status transition logic: `internal/session/todo_test.go`
- TUI tests for todos view and key bindings following existing `home_test.go` patterns
