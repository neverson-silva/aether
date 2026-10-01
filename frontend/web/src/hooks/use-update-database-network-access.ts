import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPatch } from '../api/client'
import type { Database } from '../api/types'

export function useUpdateDatabaseNetworkAccess(dbId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { public_access: boolean; external_port: number }) =>
      apiPatch<Database>(`/api/v1/databases/${dbId}/network-access`, body),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['database', dbId] }),
        queryClient.invalidateQueries({ queryKey: ['databases'] }),
      ])
    },
  })
}
