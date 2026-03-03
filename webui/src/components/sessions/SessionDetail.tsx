import { useParams, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useSession } from '@/hooks/useSessions'
import { api } from '@/api/client'
import { StatusBadge } from './StatusBadge'
import { PRBadge } from './PRBadge'
import { TerminalView } from './TerminalView'

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export function SessionDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: session, isLoading } = useSession(id!)
  const queryClient = useQueryClient()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const stopMutation = useMutation({
    mutationFn: () => api.stopSession(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['sessions'] }),
  })

  const restartMutation = useMutation({
    mutationFn: () => api.restartSession(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['sessions'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteSession(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      navigate('/sessions')
      // TUI reloads from DB via fsnotify debounce (~200ms); refetch again after
      // the delayed sessions_changed WS broadcast to pick up the removal.
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ['sessions'] }), 1000)
    },
  })

  if (isLoading || !session) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        {isLoading ? 'Loading...' : 'Session not found'}
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full bg-background">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-2.5 border-b border-border bg-card shrink-0">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="font-medium text-foreground truncate">{session.title}</h2>
            <StatusBadge status={session.status} />
            {session.pr && <PRBadge pr={session.pr} />}
          </div>
          {session.worktree_branch && (
            <div className="text-xs text-muted-foreground font-mono mt-0.5">{session.worktree_branch}</div>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={() => restartMutation.mutate()}
            disabled={restartMutation.isPending}
            className="px-2.5 py-1 rounded text-xs font-medium bg-muted hover:bg-accent text-card-foreground transition-colors disabled:opacity-50"
          >
            Restart
          </button>
          <button
            onClick={() => stopMutation.mutate()}
            disabled={stopMutation.isPending || session.status === 'stopped'}
            className="px-2.5 py-1 rounded text-xs font-medium bg-muted hover:bg-accent text-card-foreground transition-colors disabled:opacity-50"
            title="Stop tmux session (keeps session record)"
          >
            Stop
          </button>
          {confirmDelete ? (
            <div className="flex items-center gap-1">
              <span className="text-xs text-red-400">Delete?</span>
              <button
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
                className="px-2 py-1 rounded text-xs font-medium bg-red-700 hover:bg-red-600 text-white transition-colors disabled:opacity-50"
              >
                Yes
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                className="px-2 py-1 rounded text-xs font-medium bg-muted hover:bg-accent text-card-foreground transition-colors"
              >
                No
              </button>
            </div>
          ) : (
            <button
              onClick={() => setConfirmDelete(true)}
              className="px-2.5 py-1 rounded text-xs font-medium bg-accent hover:bg-red-900/50 text-muted-foreground hover:text-red-300 transition-colors"
              title="Delete session"
            >
              🗑
            </button>
          )}
        </div>
      </div>

      {/* Info bar */}
      {(session.group_path || session.project_path || session.worktree_branch) && (
        <div className="flex items-center gap-4 px-4 py-1 text-xs text-muted-foreground bg-muted/30 border-b border-border shrink-0 overflow-x-auto">
          {session.group_path && (
            <span>📁 {session.group_path}</span>
          )}
          {session.project_path && (
            <span className="font-mono truncate max-w-48" title={session.project_path}>
              {session.project_path.split('/').slice(-2).join('/')}
            </span>
          )}
          {session.worktree_branch && (
            <span className="font-mono text-blue-400">⎇ {session.worktree_branch}</span>
          )}
          <span className="ml-auto shrink-0">{relativeTime(session.created_at)}</span>
        </div>
      )}

      {/* Terminal — flex-1 fills remaining space */}
      <div className="flex-1 overflow-hidden p-1">
        <TerminalView sessionId={session.id} className="h-full" />
      </div>

    </div>
  )
}
