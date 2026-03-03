import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useSessions() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: api.getSessions,
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}

export function useSession(id: string) {
  return useQuery({
    queryKey: ['sessions', id],
    queryFn: () => api.getSession(id),
    staleTime: 10_000,
  })
}
