import { useState, lazy, Suspense, useCallback } from 'react'
import { Routes, Route, Navigate, NavLink } from 'react-router-dom'
import { SessionList } from '../sessions/SessionList'
import { CreateSessionDialog } from '../dialogs/CreateSessionDialog'
import { AddProjectDialog } from '../projects/AddProjectDialog'
import { useUIStore } from '@/stores/uiStore'
import { useSessions } from '@/hooks/useSessions'
import { usePRDashboard } from '@/hooks/usePRDashboard'
import { cn } from '@/lib/utils'

// Lazy-load heavy components so xterm.js isn't in the initial bundle
const SessionDetail = lazy(() =>
  import('../sessions/SessionDetail').then((m) => ({ default: m.SessionDetail }))
)
const TodoBoard = lazy(() =>
  import('../todos/TodoBoard').then((m) => ({ default: m.TodoBoard }))
)
const ProjectDetail = lazy(() =>
  import('../projects/ProjectDetail').then((m) => ({ default: m.ProjectDetail }))
)
const PROverview = lazy(() =>
  import('../sessions/PROverview').then((m) => ({ default: m.PROverview }))
)

const THEME_CYCLE = { dark: 'light', light: 'system', system: 'dark' } as const
const THEME_ICON = { dark: '🌙', light: '☀️', system: '💻' } as const

export function AppShell() {
  const [createOpen, setCreateOpen] = useState(false)
  const [addProjectOpen, setAddProjectOpen] = useState(false)
  const { sidebarOpen, setSidebarOpen, sidebarWidth, setSidebarWidth, theme, setTheme, selectedSessionId } = useUIStore()
  const { data: sessions = [] } = useSessions()

  // Derive the current project from the selected session's group_path so the
  // new session dialog can pre-populate the project field
  const selectedSession = sessions.find((s) => s.id === selectedSessionId)
  const defaultGroup = selectedSession?.group_path || undefined
  const { data: prDashboard } = usePRDashboard()
  const prCount = prDashboard?.all?.length ?? sessions.filter((s) => s.pr && (s.pr.state === 'OPEN' || s.pr.state === 'DRAFT')).length

  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = sidebarWidth
    const onMove = (mv: MouseEvent) => {
      const newW = Math.max(160, Math.min(480, startWidth + mv.clientX - startX))
      setSidebarWidth(newW)
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [sidebarWidth, setSidebarWidth])

  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col border-r border-border bg-card',
          'transition-[width] duration-200',
          !sidebarOpen && 'overflow-hidden'
        )}
        style={sidebarOpen ? { width: sidebarWidth } : { width: 0 }}
      >
        {/* Sidebar header */}
        <div className="flex items-center px-3 py-3 border-b border-border shrink-0">
          <span className="font-semibold text-foreground text-sm">Hangar</span>
        </div>

        {/* Nav links */}
        <div className="px-2 pt-2 pb-1 shrink-0 space-y-0.5">
          <NavLink
            to="/prs"
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 px-2 py-1.5 rounded-md text-xs font-medium transition-colors',
                isActive
                  ? 'bg-muted text-foreground'
                  : 'text-muted-foreground hover:text-card-foreground hover:bg-accent'
              )
            }
          >
            <span>⎇ PRs</span>
            {prCount > 0 && (
              <span className="ml-auto text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-accent text-foreground">
                {prCount}
              </span>
            )}
          </NavLink>
          <NavLink
            to="/todos"
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 px-2 py-1.5 rounded-md text-xs font-medium transition-colors',
                isActive
                  ? 'bg-muted text-foreground'
                  : 'text-muted-foreground hover:text-card-foreground hover:bg-accent'
              )
            }
          >
            ☑ Todos
          </NavLink>
        </div>

        {/* Session list — scrollable */}
        <SessionList />

        {/* Bottom action buttons */}
        <div className="p-2 border-t border-border shrink-0 space-y-1.5">
          <button
            onClick={() => setAddProjectOpen(true)}
            className="w-full flex items-center justify-center gap-2 rounded-md py-1.5 text-xs font-medium bg-accent hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            + New Project
          </button>
          <button
            onClick={() => setCreateOpen(true)}
            className="w-full flex items-center justify-center gap-2 rounded-md py-2 text-sm font-medium bg-muted hover:bg-accent text-foreground transition-colors"
          >
            + New Session
          </button>
        </div>
      </aside>

      {/* Drag handle */}
      {sidebarOpen && (
        <div
          onMouseDown={startResize}
          className="w-1 cursor-col-resize hover:bg-primary/50 transition-colors shrink-0"
          style={{ background: 'transparent' }}
        />
      )}

      {/* Main content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <div className="flex items-center gap-2 px-3 py-2 border-b border-border bg-card shrink-0">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
            title="Toggle sidebar"
          >
            &#9776;
          </button>
          <div className="flex-1" />
          <button
            onClick={() => setTheme(THEME_CYCLE[theme])}
            className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors text-sm"
            title={`Theme: ${theme}`}
          >
            {THEME_ICON[theme]}
          </button>
        </div>

        {/* Route content */}
        <div className="flex-1 overflow-hidden">
          <Suspense
            fallback={
              <div className="flex items-center justify-center h-full text-muted-foreground">
                Loading...
              </div>
            }
          >
            <Routes>
              <Route
                path="/sessions"
                element={
                  <div className="flex items-center justify-center h-full text-muted-foreground">
                    <div className="text-center">
                      <div className="text-4xl mb-4 opacity-30">⊡</div>
                      <p className="text-sm">Select a session from the sidebar</p>
                    </div>
                  </div>
                }
              />
              <Route path="/sessions/:id" element={<SessionDetail />} />
              <Route path="/prs" element={<PROverview />} />
              <Route path="/todos" element={<TodoBoard />} />
              <Route path="/todos/:project" element={<TodoBoard />} />
              <Route path="/projects/:name" element={<ProjectDetail />} />
              <Route path="/" element={<Navigate to="/sessions" replace />} />
              <Route path="*" element={<Navigate to="/sessions" replace />} />
            </Routes>
          </Suspense>
        </div>
      </main>

      <CreateSessionDialog open={createOpen} onOpenChange={setCreateOpen} defaultGroup={defaultGroup} />
      <AddProjectDialog open={addProjectOpen} onOpenChange={setAddProjectOpen} />
    </div>
  )
}
