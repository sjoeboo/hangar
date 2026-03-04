import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSessions } from '@/hooks/useSessions'
import { usePRDashboard } from '@/hooks/usePRDashboard'
import type { PRFullInfo } from '@/api/types'
import { StatusBadge } from './StatusBadge'
import { PRDetail } from '@/components/prs/PRDetail'
import { cn } from '@/lib/utils'

type Tab = 'all' | 'mine' | 'review_requested' | 'sessions'

const STATE_ORDER: Record<string, number> = { OPEN: 0, DRAFT: 1, MERGED: 2, CLOSED: 3 }

// Strip GHE host prefix: "ghe.example.com/owner/repo" → "owner/repo"
function stripRepoHost(repo: string): string {
  const parts = repo.split('/')
  return parts.length === 3 ? `${parts[1]}/${parts[2]}` : repo
}

// Compact age string matching TUI format
function prAgeStr(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const days = Math.floor(ms / 86_400_000)
  if (days < 1) return '<1d'
  if (days < 14) return `${days}d`
  const weeks = Math.floor(days / 7)
  if (weeks < 8) return `${weeks}w`
  const months = Math.floor(days / 30)
  if (months < 24) return `${months}mo`
  return `${Math.floor(months / 12)}y`
}

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
  const [hideDrafts, setHideDrafts] = useState(false)

  const { data: dashboard, isLoading } = usePRDashboard()

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

  const rawActivePRs = tabData[activeTab]
  const activePRs = hideDrafts
    ? rawActivePRs.filter((p) => !p.is_draft && p.state !== 'DRAFT')
    : rawActivePRs
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
          <button
            onClick={() => setHideDrafts((v) => !v)}
            className={cn(
              'ml-auto mb-1 px-2 py-1 rounded text-xs font-medium border transition-colors',
              hideDrafts
                ? 'bg-(--oasis-accent)/20 text-(--oasis-accent) border-(--oasis-accent)/30'
                : 'bg-muted text-muted-foreground border-border hover:bg-accent'
            )}
          >
            {hideDrafts ? 'Show drafts' : 'Hide drafts'}
          </button>
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
      <div className="flex-1 overflow-y-auto px-2 py-2">
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
          <table className="w-full text-xs border-separate border-spacing-0">
            <thead>
              <tr className="text-muted-foreground uppercase tracking-wider">
                <th className="text-left py-1 px-2 font-medium w-10">#</th>
                <th className="text-left py-1 px-2 font-medium w-10">Age</th>
                {showRepo && <th className="text-left py-1 px-2 font-medium w-36">Repo</th>}
                <th className="text-left py-1 px-2 font-medium">Title</th>
                <th className="text-left py-1 px-2 font-medium w-20">Checks</th>
                <th className="text-left py-1 px-2 font-medium w-24">Review</th>
                <th className="text-left py-1 px-2 font-medium w-16">State</th>
                {!showAuthor && <th className="w-4" />}
              </tr>
            </thead>
            <tbody>
              {activePRs.map((pr) => {
                const session = getSessionForPR(pr)
                const isDraft = pr.is_draft || pr.state === 'DRAFT'
                return (
                  <tr
                    key={`${pr.repo ?? 'local'}-${pr.number}`}
                    onClick={() => setSelectedPR(pr)}
                    className={cn(
                      'cursor-pointer hover:bg-accent/40 transition-colors border-b border-border/40',
                      isDraft && 'opacity-60'
                    )}
                  >
                    <td className="py-1.5 px-2 font-mono text-muted-foreground whitespace-nowrap">
                      <a
                        href={pr.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={(e) => e.stopPropagation()}
                        className="hover:underline text-(--oasis-accent)"
                      >
                        #{pr.number}
                      </a>
                    </td>
                    <td className="py-1.5 px-2 text-muted-foreground whitespace-nowrap">
                      {prAgeStr(pr.created_at)}
                    </td>
                    {showRepo && (
                      <td className="py-1.5 px-2 font-mono text-muted-foreground truncate max-w-[9rem]">
                        {pr.repo ? stripRepoHost(pr.repo) : ''}
                      </td>
                    )}
                    <td className="py-1.5 px-2 min-w-0">
                      <div className="flex items-center gap-2 min-w-0">
                        <span className={cn('truncate text-foreground', isDraft && 'italic')}>
                          {pr.title}
                        </span>
                        {session && <StatusBadge status={session.status} />}
                      </div>
                      {activeTab === 'sessions' && session && (
                        <div className="text-muted-foreground truncate">
                          {session.title}
                          {session.worktree_branch && (
                            <span className="ml-2 font-mono text-blue-400">⎇ {session.worktree_branch}</span>
                          )}
                        </div>
                      )}
                    </td>
                    <td className="py-1.5 px-2 whitespace-nowrap">
                      <ChecksBadge pr={pr} />
                    </td>
                    <td className="py-1.5 px-2 whitespace-nowrap">
                      <ReviewDecisionBadge decision={pr.review_decision} />
                    </td>
                    <td className="py-1.5 px-2 whitespace-nowrap">
                      <PRStateBadge state={pr.state} isDraft={pr.is_draft} />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
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
