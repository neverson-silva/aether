import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiDelete } from '../api/client'

export function useDeleteDatabaseBackup(serviceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (backupId: string) =>
      apiDelete(`/api/v1/services/${serviceId}/backups/${backupId}`),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: ['database-backups', 'service', serviceId],
      }),
  })
}
