import { useState } from 'react'
import { Draggable } from '@hello-pangea/dnd'
import type { Todo } from '@/api/types'
import { useDeleteTodo } from '@/hooks/useTodos'
import { EditTodoDialog } from './EditTodoDialog'
import { cn } from '@/lib/utils'

interface TodoCardProps {
  todo: Todo
  index: number
}

export function TodoCard({ todo, index }: TodoCardProps) {
  const deleteMutation = useDeleteTodo()
  const [editOpen, setEditOpen] = useState(false)

  return (
    <>
      <Draggable draggableId={todo.id} index={index}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.draggableProps}
            {...provided.dragHandleProps}
            onClick={() => setEditOpen(true)}
            className={cn(
              'rounded-md border p-3 text-sm cursor-pointer active:cursor-grabbing',
              'border-border bg-accent hover:border-ring',
              'transition-colors select-none',
              snapshot.isDragging && 'shadow-lg border-ring rotate-1'
            )}
          >
            <div className="flex items-start justify-between gap-2">
              <p className="font-medium text-foreground leading-snug">{todo.title}</p>
              <button
                onClick={(e) => { e.stopPropagation(); deleteMutation.mutate(todo.id) }}
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
          </div>
        )}
      </Draggable>
      <EditTodoDialog open={editOpen} onOpenChange={setEditOpen} todo={todo} />
    </>
  )
}
