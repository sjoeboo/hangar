# WebUI Todo Edit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to click a todo card in the WebUI Kanban board to open an edit dialog for title, description, and prompt.

**Architecture:** Extend the existing `PATCH /api/v1/todos/{id}` endpoint to carry `prompt` through the API response and request types, then add an `EditTodoDialog` React component wired into `TodoCard` via an `onClick` handler.

**Tech Stack:** Go (net/http), React 19, TypeScript, @tanstack/react-query, shadcn/ui Dialog, @hello-pangea/dnd (no changes needed)

---

### Task 1: Add `prompt` to Go API types and handler

**Files:**
- Modify: `internal/apiserver/types.go`
- Modify: `internal/apiserver/todos.go`

**Step 1: Add `prompt` to `TodoResponse` in types.go**

In `internal/apiserver/types.go`, update the `TodoResponse` struct (around line 103) to add Prompt:

```go
type TodoResponse struct {
	ID          string    `json:"id"`
	ProjectPath string    `json:"project_path"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	Status      string    `json:"status"`
	SessionID   string    `json:"session_id,omitempty"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

**Step 2: Add `Prompt` to `UpdateTodoRequest` in types.go**

Update `UpdateTodoRequest` (around line 123):

```go
type UpdateTodoRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Prompt      *string `json:"prompt,omitempty"`
	Status      *string `json:"status,omitempty"`
	SessionID   *string `json:"session_id,omitempty"`
}
```

**Step 3: Wire `Prompt` in `todos.go`**

In `todoToResponse()` (line 12), add:
```go
Prompt:      t.Prompt,
```

In `updateTodo()` (around line 148), add after the `SessionID` block:
```go
if req.Prompt != nil {
    t.Prompt = *req.Prompt
}
```

**Step 4: Build to verify**

```bash
go build ./...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add internal/apiserver/types.go internal/apiserver/todos.go
git commit -m "feat(api): expose prompt field in todo response and update request"
```

---

### Task 2: Update TypeScript types and API client

**Files:**
- Modify: `webui/src/api/types.ts`
- Modify: `webui/src/api/client.ts`
- Modify: `webui/src/hooks/useTodos.ts`

**Step 1: Add `prompt` to the `Todo` interface in `types.ts`**

In `webui/src/api/types.ts`, update the `Todo` interface (around line 97):

```typescript
export interface Todo {
  id: string
  project_path: string
  title: string
  description?: string
  prompt?: string
  status: 'todo' | 'in_progress' | 'done'
  session_id?: string
  order: number
  created_at: string
  updated_at: string
}
```

**Step 2: Add `prompt` to `updateTodo` in `client.ts`**

In `webui/src/api/client.ts`, update the `updateTodo` function signature (around line 67):

```typescript
updateTodo: (id: string, req: { status?: string; title?: string; description?: string; prompt?: string }) =>
  apiFetch<Todo>(`/api/v1/todos/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(req),
  }),
```

**Step 3: Add `prompt` to `useUpdateTodo` in `useTodos.ts`**

In `webui/src/hooks/useTodos.ts`, update the mutation type (line 17):

```typescript
mutationFn: ({ id, updates }: { id: string; updates: { status?: string; title?: string; description?: string; prompt?: string } }) =>
  api.updateTodo(id, updates),
```

**Step 4: TypeScript check**

```bash
cd webui && npx tsc --noEmit
```
Expected: no errors.

**Step 5: Commit**

```bash
git add webui/src/api/types.ts webui/src/api/client.ts webui/src/hooks/useTodos.ts
git commit -m "feat(webui): add prompt field to Todo type and updateTodo API"
```

---

### Task 3: Create `EditTodoDialog` component

**Files:**
- Create: `webui/src/components/todos/EditTodoDialog.tsx`

**Step 1: Create the component**

Create `webui/src/components/todos/EditTodoDialog.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { useUpdateTodo } from '@/hooks/useTodos'
import type { Todo } from '@/api/types'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface EditTodoDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  todo: Todo
}

export function EditTodoDialog({ open, onOpenChange, todo }: EditTodoDialogProps) {
  const [title, setTitle] = useState(todo.title)
  const [description, setDescription] = useState(todo.description ?? '')
  const [prompt, setPrompt] = useState(todo.prompt ?? '')
  const updateMutation = useUpdateTodo()

  // Reset fields whenever the dialog opens with (potentially) a new todo
  useEffect(() => {
    if (open) {
      setTitle(todo.title)
      setDescription(todo.description ?? '')
      setPrompt(todo.prompt ?? '')
    }
  }, [open, todo])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) return
    updateMutation.mutate(
      {
        id: todo.id,
        updates: {
          title: title.trim(),
          description: description.trim() || undefined,
          prompt: prompt.trim() || undefined,
        },
      },
      { onSuccess: () => onOpenChange(false) }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm bg-card border-border text-foreground">
        <DialogHeader>
          <DialogTitle>Edit Todo</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="edit-todo-title">Title</Label>
            <Input
              id="edit-todo-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="bg-accent border-border"
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="edit-todo-desc">Description (optional)</Label>
            <Input
              id="edit-todo-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Additional context..."
              className="bg-accent border-border"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="edit-todo-prompt">Prompt (optional)</Label>
            <Input
              id="edit-todo-prompt"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Sent to session on start..."
              className="bg-accent border-border"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!title.trim() || updateMutation.isPending}>
              {updateMutation.isPending ? 'Saving...' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```

**Step 2: TypeScript check**

```bash
cd webui && npx tsc --noEmit
```
Expected: no errors.

**Step 3: Commit**

```bash
git add webui/src/components/todos/EditTodoDialog.tsx
git commit -m "feat(webui): add EditTodoDialog component"
```

---

### Task 4: Wire edit dialog into `TodoCard`

**Files:**
- Modify: `webui/src/components/todos/TodoCard.tsx`

**Step 1: Update `TodoCard` to open the edit dialog on click**

Replace the entire contents of `webui/src/components/todos/TodoCard.tsx`:

```tsx
import { useState } from 'react'
import { Draggable } from '@hello-pangea/dnd'
import type { Todo } from '@/api/types'
import { useDeleteTodo } from '@/hooks/useTodos'
import { EditTodoDialog } from './EditTodoDialog'
import { cn } from '@/lib/utils'

interface TodoCardProps {
  todo: Todo
  index: number
}

export function TodoCard({ todo, index }: TodoCardProps) {
  const deleteMutation = useDeleteTodo()
  const [editOpen, setEditOpen] = useState(false)

  return (
    <>
      <Draggable draggableId={todo.id} index={index}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.draggableProps}
            {...provided.dragHandleProps}
            onClick={() => setEditOpen(true)}
            className={cn(
              'rounded-md border p-3 text-sm cursor-pointer active:cursor-grabbing',
              'border-border bg-accent hover:border-ring',
              'transition-colors select-none',
              snapshot.isDragging && 'shadow-lg border-ring rotate-1'
            )}
          >
            <div className="flex items-start justify-between gap-2">
              <p className="font-medium text-foreground leading-snug">{todo.title}</p>
              <button
                onClick={(e) => { e.stopPropagation(); deleteMutation.mutate(todo.id) }}
                className="shrink-0 text-muted-foreground hover:text-red-400 transition-colors text-xs mt-0.5"
                title="Delete todo"
              >
                ✕
              </button>
            </div>
            {todo.description && (
              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{todo.description}</p>
            )}
            {todo.session_id && (
              <div className="mt-2 text-xs text-muted-foreground font-mono truncate">
                session: {todo.session_id.slice(0, 8)}…
              </div>
            )}
          </div>
        )}
      </Draggable>
      <EditTodoDialog open={editOpen} onOpenChange={setEditOpen} todo={todo} />
    </>
  )
}
```

Key changes from original:
- `cursor-grab` → `cursor-pointer` (click intent is now edit, not just drag)
- `onClick={() => setEditOpen(true)}` on the outer card div
- `e.stopPropagation()` added to delete button's onClick so it doesn't also open the dialog
- `EditTodoDialog` rendered outside the `Draggable` (in a fragment) to avoid portaling issues with drag clones

**Step 2: TypeScript check and build**

```bash
cd webui && npx tsc --noEmit
```
Expected: no errors.

**Step 3: Manual smoke test**

```bash
cd webui && npm run dev
```

Open http://localhost:5173, navigate to Todos, select a project. Verify:
1. Clicking a card body opens the edit dialog pre-filled with title/description/prompt
2. Editing and saving updates the card
3. Cancel closes without changes
4. Clicking the ✕ delete button deletes without opening the dialog
5. Dragging a card to another column still works (drag-and-drop unaffected)

**Step 4: Commit**

```bash
git add webui/src/components/todos/TodoCard.tsx
git commit -m "feat(webui): click todo card to open edit dialog"
```

---

### Task 5: Build and verify

**Step 1: Full Go build**

```bash
go build ./...
```
Expected: no errors.

**Step 2: Go tests**

```bash
go test ./internal/apiserver/... -v
```
Expected: all pass.

**Step 3: Frontend production build**

```bash
cd webui && npm run build
```
Expected: no TypeScript errors, dist/ generated.

**Step 4: Final commit if any build artifacts changed**

```bash
git status
# Only commit if tracked files changed (e.g. dist/ if embedded)
```
