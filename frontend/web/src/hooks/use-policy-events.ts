import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'
import type { AutopilotEvent } from './types'

export function usePolicyEvents(appId: string, enabled = true) {
  return useQuery({
    queryKey: ['policy-events', appId],
    queryFn: () => apiGet<AutopilotEvent[]>(`/api/v1/apps/${appId}/policy/events`),
    enabled: enabled && !!appId,
  })
}
