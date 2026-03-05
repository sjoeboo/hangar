import { useState, useRef } from 'react'
import { Draggable } from '@hello-pangea/dnd'
import type { Todo } from '@/api/types'
import { useDeleteTodo } from '@/hooks/useTodos'
import { EditTodoDialog } from './EditTodoDialog'
import { cn } from '@/lib/utils'

const DRAG_THRESHOLD = 5

interface TodoCardProps {
  todo: Todo
  index: number
}

export function TodoCard({ todo, index }: TodoCardProps) {
  const deleteMutation = useDeleteTodo()
  const [editOpen, setEditOpen] = useState(false)
  const dragStartPos = useRef<{ x: number; y: number } | null>(null)
  const wasDragging = useRef(false)

  const handleMouseDown = (e: React.MouseEvent) => {
    dragStartPos.current = { x: e.clientX, y: e.clientY }
    wasDragging.current = false
  }

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!dragStartPos.current) return
    const dx = Math.abs(e.clientX - dragStartPos.current.x)
    const dy = Math.abs(e.clientY - dragStartPos.current.y)
    if (dx > DRAG_THRESHOLD || dy > DRAG_THRESHOLD) {
      wasDragging.current = true
    }
  }

  const handleClick = () => {
    if (wasDragging.current) return
    setEditOpen(true)
  }

  return (
    <>
      <Draggable draggableId={todo.id} index={index}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.draggableProps}
            {...provided.dragHandleProps}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onClick={handleClick}
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
                aria-label="Delete todo"
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
