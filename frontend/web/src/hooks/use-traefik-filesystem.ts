import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'
import type { TraefikFileEntry, TraefikStatus } from './types'

export function useTraefikFiles() {
  return useQuery({
    queryKey: ['traefik', 'files'],
    queryFn: async () => {
      const response = await apiGet<{ entries: TraefikFileEntry[] }>(
        '/api/v1/admin/traefik/files',
      )
      return response.entries
    },
  })
}

export function useTraefikStatus() {
  return useQuery({
    queryKey: ['traefik', 'status'],
    queryFn: () => apiGet<TraefikStatus>('/api/v1/admin/traefik/status'),
  })
}
