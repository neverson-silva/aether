import { showToast } from '@aether/elisyum-ds'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPost } from '../api/client'
import type { ServiceStatus, ServiceSummary, Stats } from '../api/types'
import { qk } from './query-keys'

export type ServiceAction = 'deploy' | 'start' | 'stop' | 'restart' | 'delete'

type ServiceActionResponse = {
  result?: unknown
  status?: ServiceStatus
  operation_id?: string | null
}

export function useServiceAction(action: ServiceAction) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (serviceId: string) =>
      apiPost<ServiceActionResponse>(`/api/v1/services/${serviceId}/${action}`),
    onSuccess: (response, serviceId) => {
      const state =
        action === 'deploy'
          ? 'deploying'
          : action === 'start' || action === 'stop'
            ? response?.status
            : ''
      if (state) {
        queryClient.setQueryData<ServiceSummary>(qk.service(serviceId), (current) =>
          current ? { ...current, status: state as ServiceStatus } : current,
        )
        queryClient.setQueryData<Stats>(qk.serviceStats(serviceId), (current) =>
          current ? { ...current, state } : current,
        )
      }
      void queryClient.invalidateQueries({ queryKey: qk.services })
      void queryClient.invalidateQueries({ queryKey: qk.service(serviceId) })
      void queryClient.invalidateQueries({ queryKey: qk.serviceStats(serviceId) })
      void queryClient.invalidateQueries({ queryKey: qk.serviceContainers(serviceId) })
      void queryClient.invalidateQueries({ queryKey: qk.serviceDeployments(serviceId) })
      const message =
        action === 'deploy'
          ? 'Deployment queued'
          : action === 'start'
            ? 'Service start requested'
            : action === 'stop'
              ? 'Service stop requested'
              : action === 'restart'
                ? 'Service restart requested'
                : 'Service deleted'
      showToast(message, action === 'deploy' ? 'info' : 'success')
    },
    onError: (error) => {
      showToast(
        error instanceof Error
          ? error.message
          : 'The service action could not be completed',
        'error',
      )
    },
  })
}
