import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiDelete } from '../api/client'
import { qk } from './query-keys'

export function useDeleteMember() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orgId, userId }: { orgId: string; userId: string }) => {
      if (!orgId) throw new Error('The current organization is unavailable.')
      return apiDelete(`/api/v1/organizations/${orgId}/members/${userId}`)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.members }),
  })
}
