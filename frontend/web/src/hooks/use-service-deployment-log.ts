import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { apiGet } from '../api/client'
import { useRealtime, useRealtimeEvent } from '../components/RealtimeProvider'
import type { DeploymentLog } from './types'

export function useServiceDeploymentLog(
  serviceId: string,
  deploymentId: string | null,
) {
  const qc = useQueryClient()
  const { connected } = useRealtime()
  const key = ['service-deploy-log', serviceId, deploymentId] as const
  const query = useQuery({
    queryKey: key,
    enabled: !!serviceId && !!deploymentId,
    queryFn: () =>
      apiGet<DeploymentLog>(
        `/api/v1/services/${serviceId}/deployments/${deploymentId}/log`,
      ),
  })

  useRealtimeEvent((ev, replay) => {
    if (!deploymentId || replay) return
    const payload =
      typeof ev.payload === 'string' ? parsePayload(ev.payload) : ev.payload
    const eventDeploymentId =
      ev.resource_id || (payload?.deployment_id as string | undefined)
    if (eventDeploymentId !== deploymentId || !ev.type.startsWith('deploy.')) return
    if (ev.type === 'deploy.build.log') {
      const line = (payload?.line as string | undefined) || ev.message
      if (!line) return
      qc.setQueryData<DeploymentLog>(key, (old) => {
        const base = old ?? { number: 0, status: 'building', error: '', content: '' }
        return { ...base, content: base.content ? base.content + '\n' + line : line }
      })
      return
    }
    void qc.invalidateQueries({ queryKey: key })
  })

  const prevConnected = useRef(connected)
  useEffect(() => {
    if (prevConnected.current === false && connected === true && deploymentId) {
      qc.invalidateQueries({ queryKey: key })
    }
    prevConnected.current = connected
  }, [connected, deploymentId, qc])

  return query
}

function parsePayload(value: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(value)
    return parsed && typeof parsed === 'object'
      ? (parsed as Record<string, unknown>)
      : {}
  } catch {
    return {}
  }
}
