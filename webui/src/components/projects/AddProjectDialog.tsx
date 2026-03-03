import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface AddProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddProjectDialog({ open, onOpenChange }: AddProjectDialogProps) {
  const [name, setName] = useState('')
  const [baseDir, setBaseDir] = useState('')
  const [baseBranch, setBaseBranch] = useState('')
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: api.createProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      onOpenChange(false)
      setName('')
      setBaseDir('')
      setBaseBranch('')
    },
  })

  const errorMessage = mutation.error instanceof Error ? mutation.error.message : null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !baseDir.trim()) return
    mutation.mutate({
      name: name.trim(),
      base_dir: baseDir.trim(),
      base_branch: baseBranch.trim() || undefined,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md bg-card border-border text-foreground">
        <DialogHeader>
          <DialogTitle>New Project</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="proj-name">Project Name</Label>
            <Input
              id="proj-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-project"
              className="bg-accent border-border"
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proj-dir">Base Directory</Label>
            <Input
              id="proj-dir"
              value={baseDir}
              onChange={(e) => setBaseDir(e.target.value)}
              placeholder="/home/user/projects/my-project"
              className="bg-accent border-border font-mono text-sm"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proj-branch">Base Branch <span className="text-muted-foreground">(optional)</span></Label>
            <Input
              id="proj-branch"
              value={baseBranch}
              onChange={(e) => setBaseBranch(e.target.value)}
              placeholder="main"
              className="bg-accent border-border font-mono text-sm"
            />
          </div>
          {errorMessage && (
            <p className="text-sm text-red-400">{errorMessage}</p>
          )}
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || !baseDir.trim() || mutation.isPending}
            >
              {mutation.isPending ? 'Creating...' : 'Create Project'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
