import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsClient } from '../api/websocket'
import type { WsSessionOutputData } from '../api/types'

export function useWebSocket() {
  const queryClient = useQueryClient()

  useEffect(() => {
    wsClient.connect()

    const offChanged = wsClient.on('sessions_changed', () => {
      void queryClient.invalidateQueries({ queryKey: ['sessions'] })
    })

    const offUpdated = wsClient.on('session_updated', () => {
      void queryClient.invalidateQueries({ queryKey: ['sessions'] })
    })

    const offCreated = wsClient.on('session_created', () => {
      void queryClient.invalidateQueries({ queryKey: ['sessions'] })
    })

    const offDeleted = wsClient.on('session_deleted', () => {
      void queryClient.invalidateQueries({ queryKey: ['sessions'] })
    })

    return () => {
      offChanged()
      offUpdated()
      offCreated()
      offDeleted()
    }
  }, [queryClient])

  return wsClient
}

export function useSessionOutput(sessionId: string, onOutput: (output: string) => void) {
  useEffect(() => {
    const off = wsClient.on('session_output', (data) => {
      const d = data as WsSessionOutputData
      if (d.session_id === sessionId) {
        onOutput(d.output)
      }
    })
    return off
  }, [sessionId, onOutput])
}
