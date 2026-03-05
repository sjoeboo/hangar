import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { PRFullInfo } from '@/api/types'
import { MarkdownContent } from './MarkdownContent'
import { PRDiff } from './PRDiff'
import { PRConversation } from './PRConversation'
import { ErrorBoundary } from './ErrorBoundary'
import { cn } from '@/lib/utils'

interface PRDetailProps {
  pr: PRFullInfo
  onClose: () => void
  onNavigateToSession?: () => void
}

type DetailTab = 'overview' | 'description' | 'diff' | 'conversation'

const REVIEW_DECISION_DISPLAY: Record<string, { label: string; className: string }> = {
  APPROVED:           { label: '✓ Approved',         className: 'text-(--oasis-green)  bg-(--oasis-green)/10  border-(--oasis-green)/30' },
  CHANGES_REQUESTED:  { label: '△ Changes Requested', className: 'text-(--oasis-yellow) bg-(--oasis-yellow)/10 border-(--oasis-yellow)/30' },
  REVIEW_REQUIRED:    { label: '? Review Required',   className: 'text-muted-foreground bg-muted border-border' },
}

const STATE_DISPLAY: Record<string, { label: string; className: string }> = {
  OPEN:   { label: 'Open',   className: 'text-(--oasis-green)  bg-(--oasis-green)/10  border-(--oasis-green)/30' },
  DRAFT:  { label: 'Draft',  className: 'text-muted-foreground bg-muted border-border' },
  MERGED: { label: 'Merged', className: 'text-(--oasis-purple) bg-(--oasis-purple)/10 border-(--oasis-purple)/30' },
  CLOSED: { label: 'Closed', className: 'text-(--oasis-red)    bg-(--oasis-red)/10    border-(--oasis-red)/30' },
}

function StateBadge({ state, isDraft }: { state: string; isDraft?: boolean }) {
  const effectiveState = isDraft ? 'DRAFT' : state
  const cfg = STATE_DISPLAY[effectiveState] ?? STATE_DISPLAY.OPEN
  return (
    <span className={cn(
      'inline-flex items-center rounded border px-2 py-0.5 text-xs font-semibold',
      cfg.className
    )}>
      {cfg.label}
    </span>
  )
}

function ChecksSummary({ pr }: { pr: PRFullInfo }) {
  if (!pr.has_checks) return null
  const failed = pr.checks_failed ?? 0
  const pending = pr.checks_pending ?? 0
  const passed = pr.checks_passed ?? 0
  return (
    <div className="flex items-center gap-2 text-xs">
      {failed > 0 && <span className="text-(--oasis-red)">✗ {failed} failed</span>}
      {pending > 0 && <span className="text-(--oasis-yellow)">● {pending} pending</span>}
      {passed > 0 && <span className="text-(--oasis-green)">✓ {passed} passed</span>}
    </div>
  )
}

function ActionModal({
  title,
  placeholder,
  requireBody,
  onConfirm,
  onCancel,
}: {
  title: string
  placeholder: string
  requireBody?: boolean
  onConfirm: (body: string) => void
  onCancel: () => void
}) {
  const [body, setBody] = useState('')
  return (
    <div className="absolute inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-10">
      <div className="w-full max-w-md bg-card border border-border rounded-xl shadow-2xl p-4 mx-4">
        <h3 className="text-sm font-semibold text-foreground mb-3">{title}</h3>
        <textarea
          autoFocus
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder={placeholder}
          className="w-full h-28 resize-none rounded-lg bg-background border border-border px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-(--oasis-accent)"
        />
        <div className="flex items-center justify-end gap-2 mt-3">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 rounded text-xs font-medium bg-muted hover:bg-accent text-muted-foreground transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm(body)}
            disabled={requireBody && !body.trim()}
            className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-accent) hover:bg-(--oasis-accent)/80 text-white transition-colors disabled:opacity-50"
          >
            Submit
          </button>
        </div>
      </div>
    </div>
  )
}

export function PRDetail({ pr, onClose, onNavigateToSession }: PRDetailProps) {
  const [activeTab, setActiveTab] = useState<DetailTab>('overview')
  const [actionModal, setActionModal] = useState<
    | { kind: 'review_approve' }
    | { kind: 'review_changes' }
    | { kind: 'comment' }
    | null
  >(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
  const queryClient = useQueryClient()

  const repo = pr.repo ?? ''
  const hasRepo = !!repo

  const { data: detail, isLoading } = useQuery({
    queryKey: ['pr-detail', repo, pr.number],
    queryFn: () => api.getPRDetail(repo, pr.number),
    enabled: hasRepo,
    staleTime: 60_000,
  })

  const reviewMutation = useMutation({
    mutationFn: ({ action, body }: { action: string; body?: string }) =>
      api.submitPRReview(repo, pr.number, action, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pr-detail', repo, pr.number] })
      queryClient.invalidateQueries({ queryKey: ['prs'] })
      setActionModal(null)
    },
    onError: (err: Error) => setMutationError(err.message),
  })

  const commentMutation = useMutation({
    mutationFn: ({ body }: { body: string }) =>
      api.addPRComment(repo, pr.number, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pr-detail', repo, pr.number] })
      setActionModal(null)
    },
    onError: (err: Error) => setMutationError(err.message),
  })

  const stateMutation = useMutation({
    mutationFn: ({ action }: { action: string }) =>
      api.changePRState(repo, pr.number, action),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pr-detail', repo, pr.number] })
      queryClient.invalidateQueries({ queryKey: ['prs'] })
    },
  })

  const effectiveState = pr.is_draft ? 'DRAFT' : pr.state
  const isOwn = pr.source === 'mine'
  const isOpen = pr.state === 'OPEN' || effectiveState === 'DRAFT'

  const tabs: { key: DetailTab; label: string }[] = [
    { key: 'overview',     label: 'Overview' },
    { key: 'description',  label: 'Description' },
    { key: 'diff',         label: `Diff${detail?.files?.length ? ` (${detail.files.length})` : ''}` },
    { key: 'conversation', label: `Conversation${detail?.comments?.length || detail?.reviews?.length ? ` (${(detail?.comments?.length ?? 0) + (detail?.reviews?.length ?? 0)})` : ''}` },
  ]

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/60 z-40"
        onClick={onClose}
      />

      {/* Drawer panel */}
      <div className="fixed inset-y-0 right-0 z-50 flex flex-col w-full max-w-2xl bg-background border-l border-border shadow-2xl">
        {/* Header */}
        <div className="shrink-0 px-4 py-3 border-b border-border bg-card">
          <div className="flex items-start gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <a
                  href={pr.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs font-mono text-(--oasis-accent) hover:underline shrink-0"
                >
                  #{pr.number}
                </a>
                <StateBadge state={pr.state} isDraft={pr.is_draft} />
                {pr.review_decision && REVIEW_DECISION_DISPLAY[pr.review_decision] && (
                  <span className={cn(
                    'inline-flex items-center rounded border px-1.5 py-0.5 text-xs font-medium',
                    REVIEW_DECISION_DISPLAY[pr.review_decision].className
                  )}>
                    {REVIEW_DECISION_DISPLAY[pr.review_decision].label}
                  </span>
                )}
              </div>
              <h2 className="mt-1 text-sm font-semibold text-foreground leading-tight">
                {pr.title}
              </h2>
              <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
                {pr.repo && <span className="font-mono">{pr.repo}</span>}
                {pr.author && <span>by {pr.author}</span>}
                {pr.head_branch && pr.base_branch && (
                  <span className="font-mono text-blue-400">
                    {pr.base_branch} ← {pr.head_branch}
                  </span>
                )}
                <ChecksSummary pr={pr} />
              </div>
            </div>
            <button
              onClick={onClose}
              className="shrink-0 p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Close"
            >
              ✕
            </button>
          </div>

          {/* Tabs */}
          <div className="flex items-center gap-0 mt-3 -mb-px">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'px-3 py-1.5 text-xs font-medium border-b-2 transition-colors',
                  activeTab === tab.key
                    ? 'border-(--oasis-accent) text-foreground'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-3 relative">
          {/* Action modal overlay (scoped to content area) */}
          {actionModal && (
            <ActionModal
              title={
                actionModal.kind === 'review_approve' ? 'Approve PR' :
                actionModal.kind === 'review_changes' ? 'Request Changes' :
                'Add Comment'
              }
              placeholder={
                actionModal.kind === 'comment'
                  ? 'Write a comment…'
                  : 'Leave a review comment (optional)…'
              }
              requireBody={actionModal.kind === 'comment'}
              onCancel={() => { setActionModal(null); setMutationError(null) }}
              onConfirm={(body) => {
                setMutationError(null)
                if (actionModal.kind === 'review_approve') {
                  reviewMutation.mutate({ action: 'approve', body: body || undefined })
                } else if (actionModal.kind === 'review_changes') {
                  reviewMutation.mutate({ action: 'request_changes', body: body || undefined })
                } else {
                  commentMutation.mutate({ body })
                }
              }}
            />
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
              Loading details…
            </div>
          )}

          {!isLoading && activeTab === 'overview' && (
            <div className="space-y-4">
              {/* Branch info */}
              {(pr.head_branch || pr.base_branch) && (
                <div className="rounded-lg border border-border bg-card p-3">
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Branch</h3>
                  <div className="font-mono text-sm text-foreground">
                    {pr.base_branch ?? 'main'} ← {pr.head_branch ?? 'feature'}
                  </div>
                  {detail?.mergeability && (
                    <div className="mt-1 text-xs text-muted-foreground">
                      Mergeability: {detail.mergeability}
                    </div>
                  )}
                </div>
              )}

              {/* Checks */}
              {pr.has_checks && (
                <div className="rounded-lg border border-border bg-card p-3">
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">CI Checks</h3>
                  <ChecksSummary pr={pr} />
                </div>
              )}

              {/* Files changed summary */}
              {detail?.files && detail.files.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-3">
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                    Files Changed ({detail.files.length})
                  </h3>
                  <div className="space-y-1">
                    {detail.files.slice(0, 8).map((f) => (
                      <div key={f.path} className="flex items-center gap-2 text-xs font-mono">
                        <span className="flex-1 truncate text-foreground/80">{f.path}</span>
                        <span className="shrink-0 text-green-400">+{f.additions}</span>
                        <span className="shrink-0 text-red-400">-{f.deletions}</span>
                      </div>
                    ))}
                    {detail.files.length > 8 && (
                      <button
                        onClick={() => setActiveTab('diff')}
                        className="text-xs text-(--oasis-accent) hover:underline"
                      >
                        …and {detail.files.length - 8} more — view diff
                      </button>
                    )}
                  </div>
                </div>
              )}

              {/* Comment count */}
              {pr.comment_count != null && pr.comment_count > 0 && (
                <div className="rounded-lg border border-border bg-card p-3">
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1">Discussion</h3>
                  <button
                    onClick={() => setActiveTab('conversation')}
                    className="text-xs text-(--oasis-accent) hover:underline"
                  >
                    {pr.comment_count} comment{pr.comment_count !== 1 ? 's' : ''} — view conversation
                  </button>
                </div>
              )}
            </div>
          )}

          {activeTab === 'description' && (
            <div className="py-2">
              {isLoading ? (
                <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
                  Loading…
                </div>
              ) : detail?.body ? (
                <MarkdownContent>{detail.body}</MarkdownContent>
              ) : (
                <div className="flex items-center justify-center py-12 text-muted-foreground text-sm italic">
                  No description
                </div>
              )}
            </div>
          )}

          {!isLoading && activeTab === 'diff' && detail && (
            <ErrorBoundary>
              <PRDiff
                files={detail.files ?? []}
                diffContent={detail.diff_content ?? ''}
                reviews={detail.reviews ?? []}
              />
            </ErrorBoundary>
          )}

          {!isLoading && activeTab === 'conversation' && detail && (
            <ErrorBoundary>
              <PRConversation
                comments={detail.comments ?? []}
                reviews={detail.reviews ?? []}
              />
            </ErrorBoundary>
          )}
        </div>

        {/* Action bar */}
        {hasRepo && (
          <div className="shrink-0 px-4 py-3 border-t border-border bg-card flex items-center gap-2 flex-wrap">
            {mutationError && (
              <p className="w-full mt-1 text-xs text-(--oasis-red)">{mutationError}</p>
            )}
            {isOpen && (
              <>
                <button
                  onClick={() => setActionModal({ kind: 'review_approve' })}
                  disabled={reviewMutation.isPending}
                  className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-green)/20 hover:bg-(--oasis-green)/30 text-(--oasis-green) border border-(--oasis-green)/30 transition-colors disabled:opacity-50"
                >
                  Approve
                </button>
                <button
                  onClick={() => setActionModal({ kind: 'review_changes' })}
                  disabled={reviewMutation.isPending}
                  className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-yellow)/20 hover:bg-(--oasis-yellow)/30 text-(--oasis-yellow) border border-(--oasis-yellow)/30 transition-colors disabled:opacity-50"
                >
                  Request Changes
                </button>
              </>
            )}
            <button
              onClick={() => setActionModal({ kind: 'comment' })}
              disabled={commentMutation.isPending}
              className="px-3 py-1.5 rounded text-xs font-medium bg-muted hover:bg-accent text-muted-foreground border border-border transition-colors disabled:opacity-50"
            >
              Add Comment
            </button>
            {isOwn && isOpen && (
              <>
                <button
                  onClick={() => stateMutation.mutate({ action: 'close' })}
                  disabled={stateMutation.isPending}
                  className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-red)/20 hover:bg-(--oasis-red)/30 text-(--oasis-red) border border-(--oasis-red)/30 transition-colors disabled:opacity-50"
                >
                  Close PR
                </button>
                {pr.is_draft ? (
                  <button
                    onClick={() => stateMutation.mutate({ action: 'ready' })}
                    disabled={stateMutation.isPending}
                    className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-accent)/20 hover:bg-(--oasis-accent)/30 text-(--oasis-accent) border border-(--oasis-accent)/30 transition-colors disabled:opacity-50"
                  >
                    Mark Ready
                  </button>
                ) : (
                  <button
                    onClick={() => stateMutation.mutate({ action: 'draft' })}
                    disabled={stateMutation.isPending}
                    className="px-3 py-1.5 rounded text-xs font-medium bg-muted hover:bg-accent text-muted-foreground border border-border transition-colors disabled:opacity-50"
                  >
                    Convert to Draft
                  </button>
                )}
              </>
            )}
            {isOwn && pr.state === 'CLOSED' && (
              <button
                onClick={() => stateMutation.mutate({ action: 'reopen' })}
                disabled={stateMutation.isPending}
                className="px-3 py-1.5 rounded text-xs font-medium bg-(--oasis-green)/20 hover:bg-(--oasis-green)/30 text-(--oasis-green) border border-(--oasis-green)/30 transition-colors disabled:opacity-50"
              >
                Reopen PR
              </button>
            )}
            {onNavigateToSession && (
              <button
                onClick={onNavigateToSession}
                className="ml-auto px-3 py-1.5 rounded text-xs font-medium bg-muted hover:bg-accent text-muted-foreground border border-border transition-colors"
              >
                Go to Session →
              </button>
            )}
          </div>
        )}
      </div>
    </>
  )
}
