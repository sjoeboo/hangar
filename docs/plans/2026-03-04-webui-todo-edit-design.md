# WebUI Todo Edit — Design

Date: 2026-03-04

## Problem

The WebUI Kanban board has no way to edit a todo after creation. The TUI exposes an edit form (title, description, prompt) triggered by pressing Enter on a selected card. The WebUI equivalent is missing entirely.

## Goal

Allow users to click a todo card in the WebUI to open an edit dialog, matching the TUI's edit functionality.

## Trigger: Click Card Body

Clicking anywhere on the card body (not the delete button) opens an edit dialog pre-filled with the todo's current values. Drag-and-drop is unaffected — `@hello-pangea/dnd` only fires drag on sustained mouse movement, so a click opens the dialog without interfering with column sorting.

## Changes

### Go — API layer (`internal/apiserver/`)

1. **`types.go`**: Add `Prompt string` to `TodoResponse`; add `Prompt *string` to `UpdateTodoRequest`.
2. **`todos.go`**: Wire `t.Prompt` into `todoToResponse()`; apply `req.Prompt` in `updateTodo()`.

### TypeScript — API types (`webui/src/api/`)

3. **`types.ts`**: Add `prompt?: string` to the `Todo` interface.
4. **`client.ts`**: Include `prompt` in the `updateTodo` API call payload.

### React — hooks (`webui/src/hooks/`)

5. **`useTodos.ts`**: Extend `useUpdateTodo`'s updates type to include `prompt?: string`.

### React — components (`webui/src/components/todos/`)

6. **`EditTodoDialog.tsx`** (new): Dialog component accepting `todo: Todo` prop. Pre-fills title, description, prompt. On submit calls `useUpdateTodo`. Resets to original values on cancel/close.
7. **`TodoCard.tsx`**: Add local `editOpen` state. Wrap card body `div` with `onClick={() => setEditOpen(true)}`. Add `stopPropagation` on delete button click. Render `<EditTodoDialog>` at bottom of component.

## Non-changes

- No changes to drag-and-drop logic, column ordering, or board layout.
- No new API endpoints — `PATCH /api/v1/todos/{id}` already exists and supports all needed fields.
- Status is not editable via the dialog (already handled by drag-and-drop).
