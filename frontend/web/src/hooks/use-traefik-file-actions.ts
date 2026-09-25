import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiPost, apiPut } from '../api/client'

export function useSaveTraefikFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { path: string; content: string }) =>
      apiPut('/api/v1/admin/traefik/file', input),
    onSuccess: (_, input) => {
      queryClient.invalidateQueries({ queryKey: ['traefik', 'file', input.path] })
      queryClient.invalidateQueries({ queryKey: ['traefik', 'files'] })
    },
  })
}

export function useDeleteTraefikFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (path: string) =>
      apiDelete(`/api/v1/admin/traefik/file?path=${encodeURIComponent(path)}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['traefik', 'files'] }),
  })
}

export function useRestartTraefik() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiPost('/api/v1/admin/traefik/restart'),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['traefik', 'status'] }),
  })
}
