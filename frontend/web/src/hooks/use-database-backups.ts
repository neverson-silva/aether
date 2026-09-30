import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'
import type { BackupJob } from '../api/types'

export function useDatabaseBackups(
  serviceId: string,
  limit = 10,
  fromDate = '',
  toDate = '',
) {
  const params = new URLSearchParams({ limit: String(limit) })
  if (fromDate) params.set('from', fromDate)
  if (toDate) params.set('to', toDate)
  return useQuery({
    queryKey: ['database-backups', 'service', serviceId, limit, fromDate, toDate],
    queryFn: async () => {
      params.set('limit', String(limit + 1))
      const response = await apiGet<BackupJob[]>(
        `/api/v1/services/${serviceId}/backups?${params.toString()}`,
      )
      return {
        items: response.slice(0, limit),
        hasMore: response.length > limit,
      }
    },
    enabled: !!serviceId,
  })
}
