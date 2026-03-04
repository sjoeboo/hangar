import { useNavigate } from 'react-router-dom'
import { useUIStore } from '@/stores/uiStore'
import type { Session } from '@/api/types'
import { StatusBadge } from './StatusBadge'
import { PRBadge } from './PRBadge'
import { cn } from '@/lib/utils'

interface SessionItemProps {
  session: Session
}

export function SessionItem({ session }: SessionItemProps) {
  const navigate = useNavigate()
  const { selectedSessionId, setSelectedSession } = useUIStore()
  const isSelected = selectedSessionId === session.id

  const handleClick = () => {
    setSelectedSession(session.id)
    navigate(`/sessions/${session.id}`)
  }

  return (
    <button
      onClick={handleClick}
      className={cn(
        'w-full text-left px-3 py-2 rounded-md transition-colors',
        'hover:bg-accent/50',
        isSelected && 'bg-accent ring-1 ring-ring'
      )}
    >
      <div className="flex items-center justify-between gap-2 min-w-0">
        <span className="truncate text-sm font-medium text-foreground">{session.title}</span>
        <div className="flex items-center gap-1.5 shrink-0">
          {session.pr && <PRBadge pr={session.pr} />}
          {session.session_type === 'tower' ? (
            <span className="text-cyan-400 font-bold text-base leading-none" title="Tower">◈</span>
          ) : (
            <StatusBadge status={session.status} />
          )}
        </div>
      </div>
      {session.worktree_branch && (
        <div className="mt-0.5 text-xs text-muted-foreground truncate font-mono">
          {session.worktree_branch}
        </div>
      )}
    </button>
  )
}
