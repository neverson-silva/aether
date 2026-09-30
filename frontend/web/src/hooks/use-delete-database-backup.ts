import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiDelete } from '../api/client'
import type { BackupJob } from '../api/types'

export function useDeleteDatabaseBackup(serviceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (backupId: string) =>
      apiDelete(`/api/v1/services/${serviceId}/backups/${backupId}`),
    onSuccess: (_data, backupId) => {
      queryClient.setQueriesData<{
        items: BackupJob[]
        hasMore: boolean
      }>(
        { queryKey: ['database-backups', 'service', serviceId] },
        (current) =>
          current
            ? {
                ...current,
                items: current.items.filter((backup) => backup.id !== backupId),
              }
            : current,
      )
      return queryClient.invalidateQueries({
        queryKey: ['database-backups', 'service', serviceId],
      })
    },
  })
}
