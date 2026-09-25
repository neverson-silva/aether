import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPost } from '../api/client'
import type { Project } from '../api/types'
import { qk } from './query-keys'

export function useCreateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: string | { name: string; description?: string }) =>
      apiPost<Project>(
        '/api/v1/projects',
        typeof input === 'string' ? { name: input } : input,
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.projects }),
  })
}
