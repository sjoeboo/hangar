import { cn } from '@/lib/utils'

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  running:  { label: 'Running',  className: 'bg-green-500/20 text-green-400 border-green-500/30' },
  waiting:  { label: 'Waiting',  className: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
  idle:     { label: 'Idle',     className: 'bg-muted text-muted-foreground border-border' },
  starting: { label: 'Starting', className: 'bg-blue-500/20 text-blue-400 border-blue-500/30' },
  stopped:  { label: 'Stopped',  className: 'bg-red-500/20 text-red-400 border-red-500/30' },
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
