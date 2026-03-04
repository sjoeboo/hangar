import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { PRDashboard } from '@/api/types'

export function usePRDashboard() {
  return useQuery<PRDashboard>({
    queryKey: ['prs'],
    queryFn: api.getPRDashboard,
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}
