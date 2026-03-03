import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: api.getProjects,
    staleTime: 0,
  })
}
