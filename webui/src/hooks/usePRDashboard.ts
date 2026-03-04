import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { PRDashboard } from '@/api/types'

export function usePRDashboard() {
  return useQuery<PRDashboard>({
    queryKey: ['prs'],
    queryFn: api.getPRDashboard,
    staleTime: 0,
    refetchInterval: 15_000,
    refetchOnWindowFocus: true,
  })
}
