import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { Session } from '@/api/types'
import { SessionItem } from '../sessions/SessionItem'
import { api } from '@/api/client'
import { cn } from '@/lib/utils'

interface ProjectSectionProps {
  name: string
  sessions: Session[]
  defaultOpen?: boolean
  projectName?: string
}

export function ProjectSection({ name, sessions, defaultOpen = true, projectName }: ProjectSectionProps) {
  const [open, setOpen] = useState(defaultOpen)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const queryClient = useQueryClient()

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteProject(projectName!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      setConfirmDelete(false)
    },
  })

  return (
    <div className="mb-2">
      <div className="group w-full flex items-center gap-1 px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        <button
          onClick={() => setOpen((o) => !o)}
          className="hover:text-card-foreground transition-colors text-[10px] shrink-0"
        >
          <span className={cn('transition-transform inline-block', open ? 'rotate-90' : '')}>&#9654;</span>
        </button>
        {projectName ? (
          <NavLink
            to={`/projects/${encodeURIComponent(name)}`}
            className={({ isActive }) =>
              cn(
                'flex-1 truncate hover:text-card-foreground transition-colors text-left',
                isActive && 'text-foreground'
              )
            }
          >
            {name}
          </NavLink>
        ) : (
          <span className="flex-1 truncate">{name}</span>
        )}
        <span className="ml-auto text-muted-foreground shrink-0">{sessions.length}</span>
        {projectName && (
          <span onClick={(e) => e.stopPropagation()} className="ml-1">
            {confirmDelete ? (
              <span className="flex items-center gap-1">
                <button
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                  className="text-red-400 hover:text-red-300 text-[10px] px-1 py-0.5 rounded bg-red-900/30"
                >
                  del
                </button>
                <button
                  onClick={() => setConfirmDelete(false)}
                  className="text-muted-foreground hover:text-foreground text-[10px] px-1 py-0.5 rounded bg-muted"
                >
                  ✕
                </button>
              </span>
            ) : (
              <button
                onClick={() => setConfirmDelete(true)}
                className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-red-400 text-[10px] transition-opacity"
              >
                🗑
              </button>
            )}
          </span>
        )}
      </div>
      {open && (
        <div className="space-y-0.5 pl-1">
          {sessions.map((s) => (
            <SessionItem key={s.id} session={s} />
          ))}
        </div>
      )}
    </div>
  )
}
