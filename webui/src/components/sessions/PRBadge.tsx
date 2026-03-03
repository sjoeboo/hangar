import type { PRInfo } from '@/api/types'
import { cn } from '@/lib/utils'

interface PRBadgeProps {
  pr: PRInfo
  className?: string
}

const STATE_CLASS: Record<string, string> = {
  OPEN:   'text-(--oasis-green)',
  DRAFT:  'text-muted-foreground',
  MERGED: 'text-(--oasis-purple)',
  CLOSED: 'text-(--oasis-red)',
}

export function PRBadge({ pr, className }: PRBadgeProps) {
  return (
    <a
      href={pr.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={(e) => e.stopPropagation()}
      className={cn(
        'inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-xs font-medium',
        'border-border bg-accent hover:bg-muted transition-colors',
        STATE_CLASS[pr.state] ?? 'text-muted-foreground',
        className
      )}
    >
      PR #{pr.number}
      {pr.has_checks && (
        <span className="ml-1">
          {pr.checks_failed ? (
            <span className="text-(--oasis-red)">{pr.checks_failed}</span>
          ) : pr.checks_pending ? (
            <span className="text-(--oasis-yellow)">{pr.checks_pending}</span>
          ) : (
            <span className="text-(--oasis-green)"></span>
          )}
        </span>
      )}
      <span className="opacity-60"></span>
    </a>
  )
}
