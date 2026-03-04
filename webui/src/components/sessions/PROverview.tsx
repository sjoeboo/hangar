import { useNavigate } from 'react-router-dom'
import { useSessions } from '@/hooks/useSessions'
import { PRBadge } from './PRBadge'
import { StatusBadge } from './StatusBadge'

const STATE_ORDER: Record<string, number> = { OPEN: 0, DRAFT: 1, MERGED: 2, CLOSED: 3 }

export function PROverview() {
  const { data: sessions = [], isLoading } = useSessions()
  const navigate = useNavigate()

  const prSessions = sessions
    .filter((s) => s.pr)
    .sort((a, b) => (STATE_ORDER[a.pr!.state] ?? 4) - (STATE_ORDER[b.pr!.state] ?? 4))

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Loading...
      </div>
    )
  }

  if (prSessions.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        <div className="text-center">
          <div className="text-4xl mb-4 opacity-30">⎇</div>
          <p className="text-sm">No pull requests</p>
          <p className="text-xs mt-1 opacity-60">Worktree sessions with PRs appear here</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <h1 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
        Pull Requests ({prSessions.length})
      </h1>
      <div className="space-y-2">
        {prSessions.map((session) => (
          <button
            key={session.id}
            onClick={() => navigate(`/sessions/${session.id}`)}
            className="w-full text-left p-3 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors"
          >
            <div className="flex items-center gap-2 min-w-0">
              <PRBadge pr={session.pr!} className="shrink-0" />
              <span className="font-medium text-sm truncate text-foreground">{session.title}</span>
              <div className="ml-auto shrink-0">
                <StatusBadge status={session.status} />
              </div>
            </div>
            <div className="mt-1 text-xs text-muted-foreground truncate">
              {session.pr!.title}
            </div>
            {(session.group_path || session.worktree_branch) && (
              <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                {session.group_path && <span>📁 {session.group_path}</span>}
                {session.worktree_branch && (
                  <span className="font-mono text-blue-400">⎇ {session.worktree_branch}</span>
                )}
              </div>
            )}
          </button>
        ))}
      </div>
    </div>
  )
}
