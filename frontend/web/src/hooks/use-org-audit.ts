import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'
import type { AuditLog } from '../api/types'

export function useOrgAudit(orgId: string) {
  return useQuery({
    queryKey: ['org-audit', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const response = await apiGet<AuditLog[] | { events?: AuditLog[] }>(
        `/api/v1/organizations/${orgId}/audit`,
      )
      return Array.isArray(response) ? response : (response.events ?? [])
    },
  })
}
