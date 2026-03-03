import { useState } from 'react'
import { useCreateTodo } from '@/hooks/useTodos'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface AddTodoDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectPath: string
}

export function AddTodoDialog({ open, onOpenChange, projectPath }: AddTodoDialogProps) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const createMutation = useCreateTodo()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) return
    createMutation.mutate(
      { project_path: projectPath, title: title.trim(), description: description.trim() || undefined },
      {
        onSuccess: () => {
          onOpenChange(false)
          setTitle('')
          setDescription('')
        },
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm bg-card border-border text-foreground">
        <DialogHeader>
          <DialogTitle>Add Todo</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="todo-title">Title</Label>
            <Input id="todo-title" value={title} onChange={(e) => setTitle(e.target.value)}
              placeholder="Implement feature X" className="bg-accent border-border" autoFocus />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="todo-desc">Description (optional)</Label>
            <Input id="todo-desc" value={description} onChange={(e) => setDescription(e.target.value)}
              placeholder="Additional context..." className="bg-accent border-border" />
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={!title.trim() || createMutation.isPending}>
              {createMutation.isPending ? 'Adding...' : 'Add Todo'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
