import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useSessions } from '@/hooks/useSessions'
import { api } from '@/api/client'
import type { PRFullInfo } from '@/api/types'
import { PRBadge } from './PRBadge'
import { StatusBadge } from './StatusBadge'
import { PRDetail } from '@/components/prs/PRDetail'
import { cn } from '@/lib/utils'

type Tab = 'all' | 'mine' | 'review_requested' | 'sessions'

const STATE_ORDER: Record<string, number> = { OPEN: 0, DRAFT: 1, MERGED: 2, CLOSED: 3 }

const REVIEW_DECISION_CONFIG: Record<string, { label: string; className: string }> = {
  APPROVED:           { label: '✓ Approved',  className: 'text-(--oasis-green)' },
  CHANGES_REQUESTED:  { label: '△ Changes',   className: 'text-(--oasis-yellow)' },
  REVIEW_REQUIRED:    { label: '? Pending',   className: 'text-muted-foreground' },
}

function ReviewDecisionBadge({ decision }: { decision?: string }) {
  if (!decision) return null
  const cfg = REVIEW_DECISION_CONFIG[decision]
  if (!cfg) return null
  return (
    <span className={cn('text-xs font-medium', cfg.className)}>
      {cfg.label}
    </span>
  )
}

function PRStateBadge({ state, isDraft }: { state: string; isDraft?: boolean }) {
  if (isDraft || state === 'DRAFT') {
    return (
      <span className="inline-flex items-center rounded border border-border px-1.5 py-0.5 text-xs font-medium text-muted-foreground bg-accent">
        Draft
      </span>
    )
  }
  const cls: Record<string, string> = {
    OPEN:   'text-(--oasis-green)',
    MERGED: 'text-(--oasis-purple)',
    CLOSED: 'text-(--oasis-red)',
  }
  return (
    <span className={cn(
      'inline-flex items-center rounded border border-border px-1.5 py-0.5 text-xs font-medium bg-accent',
      cls[state] ?? 'text-muted-foreground'
    )}>
      {state.charAt(0) + state.slice(1).toLowerCase()}
    </span>
  )
}

function ChecksBadge({ pr }: { pr: PRFullInfo }) {
  if (!pr.has_checks) return null
  return (
    <span className="flex items-center gap-0.5 text-xs">
      {(pr.checks_failed ?? 0) > 0 && (
        <span className="text-(--oasis-red)">✗{pr.checks_failed}</span>
      )}
      {(pr.checks_pending ?? 0) > 0 && (
        <span className="text-(--oasis-yellow)">●{pr.checks_pending}</span>
      )}
      {(pr.checks_passed ?? 0) > 0 && (
        <span className="text-(--oasis-green)">✓{pr.checks_passed}</span>
      )}
    </span>
  )
}

export function PROverview() {
  const { data: sessions = [] } = useSessions()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<Tab>('all')
  const [selectedPR, setSelectedPR] = useState<PRFullInfo | null>(null)

  const { data: dashboard, isLoading } = useQuery({
    queryKey: ['prs'],
    queryFn: api.getPRDashboard,
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  // Build the sessions tab PRs from session data + enrich with dashboard sessions map
  const sessionPRs: PRFullInfo[] = sessions
    .filter((s) => s.pr)
    .map((s) => {
      const dashPR = dashboard?.sessions?.[s.id]
      if (dashPR) return dashPR
      // Fallback: wrap PRInfo as PRFullInfo
      return {
        ...s.pr!,
        created_at: s.created_at,
        updated_at: s.last_accessed_at ?? s.created_at,
        source: 'session',
        session_id: s.id,
      } as PRFullInfo
    })
    .sort((a, b) => (STATE_ORDER[a.state] ?? 4) - (STATE_ORDER[b.state] ?? 4))

  const allPRs = (dashboard?.all ?? []).sort(
    (a, b) => (STATE_ORDER[a.state] ?? 4) - (STATE_ORDER[b.state] ?? 4)
  )
  const minePRs = (dashboard?.mine ?? []).sort(
    (a, b) => (STATE_ORDER[a.state] ?? 4) - (STATE_ORDER[b.state] ?? 4)
  )
  const reviewPRs = (dashboard?.review_requested ?? []).sort(
    (a, b) => (STATE_ORDER[a.state] ?? 4) - (STATE_ORDER[b.state] ?? 4)
  )

  const tabData: Record<Tab, PRFullInfo[]> = {
    all: allPRs,
    mine: minePRs,
    review_requested: reviewPRs,
    sessions: sessionPRs,
  }

  const tabs: { key: Tab; label: string; count: number }[] = [
    { key: 'all',              label: 'All',             count: allPRs.length },
    { key: 'mine',             label: 'Mine',            count: minePRs.length },
    { key: 'review_requested', label: 'Review Requests', count: reviewPRs.length },
    { key: 'sessions',         label: 'Sessions',        count: sessionPRs.length },
  ]

  const activePRs = tabData[activeTab]
  const showRepo = activeTab !== 'sessions'
  const showAuthor = activeTab !== 'mine'

  const getSessionForPR = (pr: PRFullInfo) => {
    if (!pr.session_id) return null
    return sessions.find((s) => s.id === pr.session_id) ?? null
  }

  if (isLoading && sessionPRs.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Loading...
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header + Tabs */}
      <div className="shrink-0 px-4 pt-4 pb-0 border-b border-border">
        <h1 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
          Pull Requests
        </h1>
        <div className="flex items-center gap-0 -mb-px">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'px-3 py-1.5 text-xs font-medium border-b-2 transition-colors whitespace-nowrap',
                activeTab === tab.key
                  ? 'border-(--oasis-accent) text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              )}
            >
              {tab.label}
              {tab.count > 0 && (
                <span className={cn(
                  'ml-1.5 rounded-full px-1.5 py-0.5 text-[10px] font-semibold',
                  activeTab === tab.key ? 'bg-(--oasis-accent)/20 text-(--oasis-accent)' : 'bg-muted text-muted-foreground'
                )}>
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* PR list */}
      <div className="flex-1 overflow-y-auto p-4">
        {activePRs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <div className="text-4xl mb-4 opacity-30">⎇</div>
              <p className="text-sm">No pull requests</p>
              <p className="text-xs mt-1 opacity-60">
                {activeTab === 'sessions'
                  ? 'Worktree sessions with PRs appear here'
                  : activeTab === 'mine'
                  ? 'Your open PRs appear here'
                  : activeTab === 'review_requested'
                  ? 'PRs awaiting your review appear here'
                  : 'No PRs found'}
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            {activePRs.map((pr) => {
              const session = getSessionForPR(pr)
              return (
                <button
                  key={`${pr.repo ?? 'local'}-${pr.number}`}
                  onClick={() => setSelectedPR(pr)}
                  className="w-full text-left p-3 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <a
                      href={pr.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={(e) => e.stopPropagation()}
                      className="shrink-0"
                    >
                      <PRBadge pr={pr} />
                    </a>
                    <span className="font-medium text-sm truncate text-foreground">
                      {pr.title}
                    </span>
                    <div className="ml-auto shrink-0 flex items-center gap-2">
                      <ReviewDecisionBadge decision={pr.review_decision} />
                      <PRStateBadge state={pr.state} isDraft={pr.is_draft} />
                      {session && <StatusBadge status={session.status} />}
                    </div>
                  </div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
                    {showRepo && pr.repo && (
                      <span className="font-mono">{pr.repo}</span>
                    )}
                    {showAuthor && pr.author && (
                      <span>by {pr.author}</span>
                    )}
                    {pr.head_branch && pr.base_branch && (
                      <span className="font-mono text-blue-400">
                        {pr.base_branch} ← {pr.head_branch}
                      </span>
                    )}
                    <ChecksBadge pr={pr} />
                    {pr.comment_count != null && pr.comment_count > 0 && (
                      <span>{pr.comment_count} comment{pr.comment_count !== 1 ? 's' : ''}</span>
                    )}
                  </div>
                  {activeTab === 'sessions' && session && (
                    <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="font-medium text-foreground/70">{session.title}</span>
                      {session.group_path && <span>📁 {session.group_path}</span>}
                      {session.worktree_branch && (
                        <span className="font-mono text-blue-400">⎇ {session.worktree_branch}</span>
                      )}
                    </div>
                  )}
                </button>
              )
            })}
          </div>
        )}
      </div>

      {/* PR Detail modal */}
      {selectedPR && (
        <PRDetail
          pr={selectedPR}
          onClose={() => setSelectedPR(null)}
          onNavigateToSession={
            selectedPR.session_id
              ? () => { navigate(`/sessions/${selectedPR.session_id}`); setSelectedPR(null) }
              : undefined
          }
        />
      )}
    </div>
  )
}
