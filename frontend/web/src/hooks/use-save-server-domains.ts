import { useMutation } from '@tanstack/react-query'
import { apiPut } from '../api/client'
import type { ServerDomainSettings } from './use-server-domains'

export function useSaveServerDomains() {
  return useMutation({
    mutationFn: (settings: ServerDomainSettings) =>
      apiPut<ServerDomainSettings>('/api/v1/admin/server-domains', settings),
  })
}
