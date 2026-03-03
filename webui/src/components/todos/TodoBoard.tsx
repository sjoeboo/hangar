import { useState, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { DragDropContext, Droppable, type DropResult } from '@hello-pangea/dnd'
import { useTodos, useUpdateTodo } from '@/hooks/useTodos'
import { useProjects } from '@/hooks/useProjects'
import { useSessions } from '@/hooks/useSessions'
import { TodoCard } from './TodoCard'
import { AddTodoDialog } from './AddTodoDialog'
import type { Todo } from '@/api/types'
import { cn } from '@/lib/utils'

const COLUMNS: { key: Todo['status']; label: string; color: string }[] = [
  { key: 'todo',        label: 'Todo',        color: 'border-border' },
  { key: 'in_progress', label: 'In Progress', color: 'border-blue-600'  },
  { key: 'done',        label: 'Done',        color: 'border-green-600' },
]

export function TodoBoard() {
  const { project: projectParam } = useParams<{ project?: string }>()
  const { data: projects = [] } = useProjects()
  const { data: sessions = [] } = useSessions()
  const [selectedProject, setSelectedProject] = useState(projectParam ?? '')
  const [addOpen, setAddOpen] = useState(false)

  // Build a deduplicated list of project paths using base_dir from projects config
  // (primary source, matches what the TUI uses as the todo key) plus non-worktree
  // session paths as a fallback for sessions not tied to a project config entry.
  // Worktree session paths (containing /.worktrees/) are excluded because they are
  // subdirectories of the repo root and would never match a todo's project_path.
  const projectOptions = useMemo(() => {
    const opts = new Map<string, string>() // path → display label
    for (const p of projects) {
      if (p.base_dir) opts.set(p.base_dir, p.name)
    }
    for (const s of sessions) {
      const path = s.project_path ?? ''
      if (path && !path.includes('/.worktrees/') && !opts.has(path)) {
        opts.set(path, path.split('/').pop() ?? path)
      }
    }
    return Array.from(opts.entries()).map(([path, label]) => ({ path, label }))
  }, [projects, sessions])

  const { data: todos = [] } = useTodos(selectedProject)
  const updateMutation = useUpdateTodo()

  const onDragEnd = (result: DropResult) => {
    if (!result.destination) return
    const { draggableId, destination } = result
    const newStatus = destination.droppableId as Todo['status']
    updateMutation.mutate({ id: draggableId, updates: { status: newStatus } })
  }

  const grouped = COLUMNS.reduce((acc, col) => {
    acc[col.key] = todos.filter((t) => t.status === col.key)
    return acc
  }, {} as Record<string, Todo[]>)

  return (
    <div className="flex flex-col h-full bg-background">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-2.5 border-b border-border bg-card shrink-0">
        <h2 className="font-medium text-foreground">Todos</h2>
        <select
          value={selectedProject}
          onChange={(e) => setSelectedProject(e.target.value)}
          className="rounded-md border border-border bg-accent px-2 py-1 text-sm text-foreground"
        >
          <option value="">— select project —</option>
          {projectOptions.map(({ path, label }) => (
            <option key={path} value={path}>{label}</option>
          ))}
        </select>
        <button
          onClick={() => setAddOpen(true)}
          className="ml-auto px-3 py-1 rounded-md text-sm font-medium bg-muted hover:bg-accent text-foreground transition-colors"
        >
          + Add Todo
        </button>
      </div>

      {/* Kanban board */}
      {!selectedProject && (
        <div className="flex items-center justify-center flex-1 text-muted-foreground text-sm">
          Select a project above to view todos
        </div>
      )}
      <DragDropContext onDragEnd={onDragEnd}>
        <div className={cn('flex gap-4 flex-1 overflow-hidden p-4', !selectedProject && 'hidden')}>
          {COLUMNS.map((col) => (
            <div key={col.key} className="flex flex-col flex-1 min-w-0">
              {/* Column header */}
              <div className={cn('flex items-center justify-between mb-3 pb-2 border-b-2', col.color)}>
                <span className="text-sm font-semibold text-card-foreground">{col.label}</span>
                <span className="text-xs text-muted-foreground bg-accent rounded-full px-2 py-0.5">
                  {grouped[col.key]?.length ?? 0}
                </span>
              </div>

              {/* Droppable column */}
              <Droppable droppableId={col.key}>
                {(provided, snapshot) => (
                  <div
                    ref={provided.innerRef}
                    {...provided.droppableProps}
                    className={cn(
                      'flex-1 space-y-2 overflow-y-auto rounded-md p-1 transition-colors',
                      snapshot.isDraggingOver && 'bg-card/50'
                    )}
                  >
                    {(grouped[col.key] ?? []).map((todo, i) => (
                      <TodoCard key={todo.id} todo={todo} index={i} />
                    ))}
                    {provided.placeholder}
                    {(grouped[col.key] ?? []).length === 0 && !snapshot.isDraggingOver && (
                      <div className="text-center text-muted-foreground text-xs py-8">
                        Drop here
                      </div>
                    )}
                  </div>
                )}
              </Droppable>
            </div>
          ))}
        </div>
      </DragDropContext>

      <AddTodoDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        projectPath={selectedProject}
      />
    </div>
  )
}
