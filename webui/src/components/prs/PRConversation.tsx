import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { PRComment, PRReview } from '@/api/types'
import { cn } from '@/lib/utils'

interface PRConversationProps {
  comments: PRComment[]
  reviews: PRReview[]
}

const REVIEW_STATE_CONFIG: Record<string, { label: string; className: string }> = {
  APPROVED:           { label: 'Approved',         className: 'text-(--oasis-green)  bg-(--oasis-green)/10  border-(--oasis-green)/30' },
  CHANGES_REQUESTED:  { label: 'Changes Requested', className: 'text-(--oasis-yellow) bg-(--oasis-yellow)/10 border-(--oasis-yellow)/30' },
  COMMENTED:          { label: 'Commented',         className: 'text-muted-foreground bg-muted border-border' },
  DISMISSED:          { label: 'Dismissed',         className: 'text-muted-foreground bg-muted border-border' },
}

function avatarColor(author: string): string {
  const colors = [
    'bg-blue-500', 'bg-purple-500', 'bg-green-600', 'bg-yellow-600',
    'bg-red-500', 'bg-pink-500', 'bg-indigo-500', 'bg-teal-600',
  ]
  let hash = 0
  for (let i = 0; i < author.length; i++) {
    hash = (hash * 31 + author.charCodeAt(i)) & 0xffff
  }
  return colors[hash % colors.length]
}

function Avatar({ author }: { author: string }) {
  return (
    <div className={cn(
      'shrink-0 w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold text-white uppercase',
      avatarColor(author)
    )}>
      {author.charAt(0)}
    </div>
  )
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

// Minimal markdown: bold, italic, inline code, code blocks
function renderMarkdown(text: string): string {
  // Escape HTML first
  let out = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // Code blocks
  out = out.replace(/```[\s\S]*?```/g, (match) => {
    const inner = match.slice(3, -3).replace(/^\w*\n/, '')
    return `<pre class="my-1 rounded bg-muted px-2 py-1 text-xs font-mono overflow-x-auto whitespace-pre-wrap">${inner}</pre>`
  })

  // Inline code
  out = out.replace(/`([^`]+)`/g, '<code class="rounded bg-muted px-1 text-xs font-mono">$1</code>')

  // Bold
  out = out.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  out = out.replace(/__([^_]+)__/g, '<strong>$1</strong>')

  // Italic
  out = out.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  out = out.replace(/_([^_]+)_/g, '<em>$1</em>')

  // Newlines
  out = out.replace(/\n/g, '<br/>')

  return out
}

type TimelineItem =
  | { kind: 'comment'; item: PRComment; time: string }
  | { kind: 'review'; item: PRReview; time: string }

export function PRConversation({ comments, reviews }: PRConversationProps) {
  // Only PR-level comments (no path)
  const prComments = comments.filter((c) => !c.path)

  // Build timeline sorted by time
  const timeline: TimelineItem[] = [
    ...prComments.map((c) => ({ kind: 'comment' as const, item: c, time: c.created_at })),
    ...reviews.map((r) => ({ kind: 'review' as const, item: r, time: r.created_at })),
  ].sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime())

  if (timeline.length === 0) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
        No comments or reviews yet
      </div>
    )
  }

  return (
    <div className="space-y-4 py-2">
      {timeline.map((entry, idx) => {
        if (entry.kind === 'comment') {
          const comment = entry.item as PRComment
          return (
            <div key={`comment-${comment.id}`} className="flex gap-3">
              <Avatar author={comment.author} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-medium text-foreground">{comment.author}</span>
                  <span className="text-xs text-muted-foreground">{relativeTime(comment.created_at)}</span>
                </div>
                <div className="prose prose-sm prose-invert max-w-none text-sm">
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{comment.body}</ReactMarkdown>
                </div>
              </div>
            </div>
          )
        } else {
          const review = entry.item as PRReview
          const cfg = REVIEW_STATE_CONFIG[review.state] ?? REVIEW_STATE_CONFIG.COMMENTED
          return (
            <div key={`review-${idx}`} className="flex gap-3">
              <Avatar author={review.author} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1 flex-wrap">
                  <span className="text-xs font-medium text-foreground">{review.author}</span>
                  <span className={cn(
                    'inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold',
                    cfg.className
                  )}>
                    {cfg.label}
                  </span>
                  <span className="text-xs text-muted-foreground">{relativeTime(review.created_at)}</span>
                </div>
                {review.body && (
                  <div className="prose prose-sm prose-invert max-w-none text-sm mb-2">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{review.body}</ReactMarkdown>
                  </div>
                )}
                {(review.comments?.length ?? 0) > 0 && (
                  <div className="ml-2 pl-3 border-l border-border space-y-2">
                    <p className="text-xs text-muted-foreground">{review.comments!.length} inline comment{review.comments!.length !== 1 ? 's' : ''}</p>
                    {review.comments!.slice(0, 3).map((c) => (
                      <div key={c.id} className="text-xs text-muted-foreground">
                        {c.path && <span className="font-mono text-blue-400">{c.path}{c.line ? `:${c.line}` : ''}</span>}
                        {c.path && ' — '}
                        <span className="truncate">{c.body.slice(0, 80)}{c.body.length > 80 ? '…' : ''}</span>
                      </div>
                    ))}
                    {review.comments!.length > 3 && (
                      <p className="text-xs text-muted-foreground">…and {review.comments!.length - 3} more</p>
                    )}
                  </div>
                )}
              </div>
            </div>
          )
        }
      })}
    </div>
  )
}
