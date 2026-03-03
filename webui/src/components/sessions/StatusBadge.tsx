import { cn } from '@/lib/utils'

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  running:  { label: 'Running',  className: 'bg-(--status-running-bg)  text-(--status-running-fg)  border-(--status-running-border)'  },
  waiting:  { label: 'Waiting',  className: 'bg-(--status-waiting-bg)  text-(--status-waiting-fg)  border-(--status-waiting-border)'  },
  starting: { label: 'Starting', className: 'bg-(--status-starting-bg) text-(--status-starting-fg) border-(--status-starting-border)' },
  stopped:  { label: 'Stopped',  className: 'bg-(--status-stopped-bg)  text-(--status-stopped-fg)  border-(--status-stopped-border)'  },
  idle:     { label: 'Idle',     className: 'bg-muted text-muted-foreground border-border' },
  unknown:  { label: 'Unknown',  className: 'bg-muted text-muted-foreground border-border' },
}

interface StatusBadgeProps {
  status: string
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const cfg = STATUS_CONFIG[status] ?? STATUS_CONFIG.unknown
  return (
    <span className={cn(
      'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
      cfg.className,
      className
    )}>
      {cfg.label}
    </span>
  )
}
