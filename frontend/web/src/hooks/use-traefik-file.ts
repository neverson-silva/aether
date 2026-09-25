import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'
import type { TraefikFile } from './types'

export function useTraefikFile(path: string) {
  return useQuery({
    queryKey: ['traefik', 'file', path],
    queryFn: () =>
      apiGet<TraefikFile>(
        `/api/v1/admin/traefik/file?path=${encodeURIComponent(path)}`,
      ),
    enabled: Boolean(path),
  })
}
