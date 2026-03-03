import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useProjects } from '@/hooks/useProjects'
import { useSessions } from '@/hooks/useSessions'
import { useTodos } from '@/hooks/useTodos'
import { api } from '@/api/client'
import { AddTodoDialog } from '../todos/AddTodoDialog'
import type { Todo, Session } from '@/api/types'

const COLUMNS = ['todo', 'in_progress', 'done'] as const
const LABELS: Record<string, string> = { todo: 'Todo', in_progress: 'In Progress', done: 'Done' }

interface SimpleTodoCardProps {
  todo: Todo
  sessions: Session[]
}

function SimpleTodoCard({ todo }: SimpleTodoCardProps) {
  const queryClient = useQueryClient()
  const deleteMutation = useMutation({
    mutationFn: () => api.deleteTodo(todo.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['todos'] }),
  })
  const updateMutation = useMutation({
    mutationFn: (status: string) => api.updateTodo(todo.id, { status }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['todos'] }),
  })

  const nextStatus: Record<string, string> = {
    todo: 'in_progress',
    in_progress: 'done',
    done: 'todo',
  }

  return (
    <div className="rounded-md border p-3 text-sm border-border bg-accent hover:border-ring transition-colors">
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium text-foreground leading-snug">{todo.title}</p>
        <button
          onClick={() => deleteMutation.mutate()}
          className="shrink-0 text-muted-foreground hover:text-red-400 transition-colors text-xs mt-0.5"
          title="Delete todo"
        >
          ✕
        </button>
      </div>
      {todo.description && (
        <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{todo.description}</p>
      )}
      {todo.session_id && (
        <div className="mt-2 text-xs text-muted-foreground font-mono truncate">
          session: {todo.session_id.slice(0, 8)}…
        </div>
      )}
      <button
        onClick={() => updateMutation.mutate(nextStatus[todo.status] ?? 'todo')}
        disabled={updateMutation.isPending}
        className="mt-2 text-[10px] text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
      >
        → move
      </button>
    </div>
  )
}

export function ProjectDetail() {
  const { name } = useParams<{ name: string }>()
  const { data: projects = [] } = useProjects()
  const { data: sessions = [] } = useSessions()
  const [addOpen, setAddOpen] = useState(false)

  const project = projects.find((p) => p.name === name)
  const projectSessions = sessions.filter(
    (s) => s.group_path === name || s.group_path === project?.name
  )
  const { data: todos = [] } = useTodos(project?.base_dir ?? '')

  if (!project) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Project not found
      </div>
    )
  }

  const statusCounts = projectSessions.reduce((acc, s) => {
    acc[s.status] = (acc[s.status] ?? 0) + 1
    return acc
  }, {} as Record<string, number>)

  return (
    <div className="flex flex-col h-full overflow-auto p-6 bg-background">
      <h1 className="text-xl font-semibold text-foreground mb-1">{project.name}</h1>
      <p className="text-xs font-mono text-muted-foreground mb-1">{project.base_dir}</p>
      {project.base_branch && (
        <p className="text-xs text-muted-foreground mb-3">
          Branch: <span className="font-mono">{project.base_branch}</span>
        </p>
      )}

      {/* Session summary */}
      <div className="flex gap-3 mb-6 flex-wrap">
        <span className="text-xs text-muted-foreground">
          {projectSessions.length} session{projectSessions.length !== 1 ? 's' : ''}
        </span>
        {Object.entries(statusCounts).map(([st, n]) => (
          <span key={st} className="text-xs px-2 py-0.5 rounded-full bg-accent text-card-foreground">
            {n} {st}
          </span>
        ))}
      </div>

      {/* Todos */}
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-semibold text-foreground">Todos</h2>
        <button
          onClick={() => setAddOpen(true)}
          className="text-xs px-2 py-1 rounded bg-accent hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
        >
          + Add
        </button>
      </div>
      {todos.length === 0 ? (
        <p className="text-xs text-muted-foreground">No todos yet.</p>
      ) : (
        <div className="grid grid-cols-3 gap-4">
          {COLUMNS.map((col) => (
            <div key={col} className="bg-card rounded-lg p-3 border border-border">
              <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                {LABELS[col]} ({todos.filter((t) => t.status === col).length})
              </h3>
              <div className="space-y-2">
                {todos
                  .filter((t) => t.status === col)
                  .map((t) => (
                    <SimpleTodoCard key={t.id} todo={t} sessions={projectSessions} />
                  ))}
              </div>
            </div>
          ))}
        </div>
      )}
      <AddTodoDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        projectPath={project.base_dir}
      />
    </div>
  )
}
