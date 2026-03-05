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
