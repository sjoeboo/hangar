import type { PRInfo } from '@/api/types'
import { cn } from '@/lib/utils'

interface PRBadgeProps {
  pr: PRInfo
  className?: string
}

const STATE_CLASS: Record<string, string> = {
  OPEN:   'text-green-400',
  DRAFT:  'text-muted-foreground',
  MERGED: 'text-purple-400',
  CLOSED: 'text-red-400',
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
            <span className="text-red-400">{pr.checks_failed}</span>
          ) : pr.checks_pending ? (
            <span className="text-yellow-400">{pr.checks_pending}</span>
          ) : (
            <span className="text-green-400"></span>
          )}
        </span>
      )}
      <span className="opacity-60"></span>
    </a>
  )
}
