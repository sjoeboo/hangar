import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useProjects } from '@/hooks/useProjects'
import { api } from '@/api/client'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface CreateSessionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  defaultGroup?: string  // pre-select this project name when dialog opens
}

export function CreateSessionDialog({ open, onOpenChange, defaultGroup }: CreateSessionDialogProps) {
  const [title, setTitle] = useState('')
  const [path, setPath] = useState('')
  const [tool, setTool] = useState('claude')
  const [group, setGroup] = useState(defaultGroup ?? '')
  const [message, setMessage] = useState('')
  const [worktree, setWorktree] = useState(true)
  const [branch, setBranch] = useState('')
  const [skipPermissions, setSkipPermissions] = useState(false)

  const { data: projects = [] } = useProjects()
  const queryClient = useQueryClient()

  // When the dialog opens, pre-select defaultGroup if provided
  useEffect(() => {
    if (open && defaultGroup) {
      setGroup(defaultGroup)
    }
  }, [open, defaultGroup])

  // Resolve selected project — used to auto-fill path and show worktree toggle
  const selectedProject = projects.find((p) => p.name === group)
  const hasProjectPath = !!selectedProject?.base_dir
  // Effective path: manual entry takes precedence, otherwise use project's base_dir
  const effectivePath = path.trim() || selectedProject?.base_dir || ''

  const [errorMsg, setErrorMsg] = useState('')

  const mutation = useMutation({
    mutationFn: api.createSession,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      // The TUI reloads sessions from DB via fsnotify debounce (~200ms).
      // Do a second refetch after 1.5s so the new session appears in the list.
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ['sessions'] }), 1500)
      onOpenChange(false)
      setTitle('')
      setPath('')
      setMessage('')
      setGroup(defaultGroup ?? '')
      setWorktree(true)
      setBranch('')
      setSkipPermissions(false)
      setErrorMsg('')
    },
    onError: (err: Error) => {
      setErrorMsg(err.message || 'Failed to create session')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim() || !effectivePath) return
    mutation.mutate({
      title: title.trim(),
      path: effectivePath,
      tool,
      group: group || undefined,
      message: message || undefined,
      worktree: (hasProjectPath && worktree) || undefined,
      branch: branch.trim() || undefined,
      skip_permissions: skipPermissions || undefined,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md bg-card border-border text-foreground">
        <DialogHeader>
          <DialogTitle>New Session</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="title">Title</Label>
            <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)}
              placeholder="My feature" className="bg-accent border-border" />
          </div>
          {/* Working dir — hidden when a project with base_dir is selected (auto-filled) */}
          {!hasProjectPath && (
            <div className="space-y-1.5">
              <Label htmlFor="path">Working Directory</Label>
              <Input id="path" value={path} onChange={(e) => setPath(e.target.value)}
                placeholder="/home/user/project" className="bg-accent border-border font-mono text-sm" />
            </div>
          )}
          {hasProjectPath && effectivePath && (
            <p className="text-xs text-muted-foreground font-mono">{effectivePath}</p>
          )}
          <div className="space-y-1.5">
            <Label htmlFor="tool">Tool</Label>
            <select id="tool" value={tool} onChange={(e) => setTool(e.target.value)}
              className="w-full rounded-md border border-border bg-accent px-3 py-2 text-sm text-foreground">
              <option value="claude">claude</option>
              <option value="shell">shell</option>
            </select>
          </div>
          {projects.length > 0 && (
            <div className="space-y-1.5">
              <Label htmlFor="group">Project (optional)</Label>
              <select id="group" value={group} onChange={(e) => setGroup(e.target.value)}
                className="w-full rounded-md border border-border bg-accent px-3 py-2 text-sm text-foreground">
                <option value="">None</option>
                {projects.map((p) => (
                  <option key={p.name} value={p.name}>{p.name}</option>
                ))}
              </select>
            </div>
          )}

          {/* Worktree toggle — only when a project with a path is selected */}
          {hasProjectPath && (
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="worktree"
                checked={worktree}
                onChange={(e) => setWorktree(e.target.checked)}
                className="w-4 h-4"
              />
              <label htmlFor="worktree" className="text-sm text-card-foreground">
                Create in worktree
              </label>
            </div>
          )}

          {/* Branch name — only when worktree is enabled */}
          {worktree && (
            <div className="flex flex-col gap-1">
              <label className="text-sm text-muted-foreground">Branch name</label>
              <input
                type="text"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder={title ? title.toLowerCase().replace(/\s+/g, '-') : 'feature-branch'}
                className="px-3 py-2 rounded bg-accent text-foreground border border-border text-sm"
              />
            </div>
          )}

          {/* Skip permissions checkbox */}
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="skipPerms"
              checked={skipPermissions}
              onChange={(e) => setSkipPermissions(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="skipPerms" className="text-sm text-amber-400">
              Skip permissions (--dangerously-skip-permissions)
            </label>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="message">Initial message (optional)</Label>
            <Input id="message" value={message} onChange={(e) => setMessage(e.target.value)}
              placeholder="Start working on..." className="bg-accent border-border" />
          </div>
          {errorMsg && (
            <p className="text-sm text-red-400">{errorMsg}</p>
          )}
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={!title.trim() || !effectivePath || mutation.isPending}>
              {mutation.isPending ? 'Creating...' : 'Create Session'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
