import { useSessions } from '@/hooks/useSessions'
import { useProjects } from '@/hooks/useProjects'
import { useWebSocket } from '@/hooks/useWebSocket'
import { ProjectSection } from '../projects/ProjectSection'
import type { Session } from '@/api/types'

export function SessionList() {
  useWebSocket() // sets up cache invalidation
  const { data: sessions = [], isLoading } = useSessions()
  const { data: projects = [] } = useProjects()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
        Loading sessions...
      </div>
    )
  }

  // Group sessions by project path
  const grouped = new Map<string, Session[]>()
  const ungrouped: Session[] = []

  for (const s of sessions) {
    if (s.group_path) {
      const key = s.group_path
      if (!grouped.has(key)) grouped.set(key, [])
      grouped.get(key)!.push(s)
    } else {
      ungrouped.push(s)
    }
  }

  // Sort project sections by project order if available
  const projectOrder = new Map(projects.map((p, i) => [p.name, p.order ?? i]))
  const sortedGroups = [...grouped.entries()].sort(([a], [b]) => {
    const aName = a.split('/').pop() ?? a
    const bName = b.split('/').pop() ?? b
    return (projectOrder.get(aName) ?? 999) - (projectOrder.get(bName) ?? 999)
  })

  // Include projects from config that have no sessions yet
  const shownGroupNames = new Set(sortedGroups.map(([path]) => path.split('/').pop() ?? path))
  const emptyProjects = projects.filter((p) => !shownGroupNames.has(p.name))
  const sortedEmptyProjects = [...emptyProjects].sort(
    (a, b) => (a.order ?? 999) - (b.order ?? 999)
  )

  return (
    <div className="overflow-y-auto flex-1 px-2 py-2">
      {sortedGroups.map(([path, groupSessions]) => (
        <ProjectSection
          key={path}
          name={path.split('/').pop() ?? path}
          projectName={path.split('/').pop() ?? path}
          sessions={groupSessions}
        />
      ))}
      {sortedEmptyProjects.map((p) => (
        <ProjectSection key={p.name} name={p.name} projectName={p.name} sessions={[]} />
      ))}
      {ungrouped.length > 0 && (
        <ProjectSection name="Ungrouped" sessions={ungrouped} />
      )}
      {sessions.length === 0 && projects.length === 0 && (
        <div className="text-center text-muted-foreground text-sm py-8">
          No sessions yet.
          <br />
          <span className="text-muted-foreground">Create one with the + button.</span>
        </div>
      )}
    </div>
  )
}
