import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { apiGet } from '../api/client'
import { useRealtime, useRealtimeEvent } from '../components/RealtimeProvider'
import type { DeploymentLog } from './types'

export function useDeploymentLog(appID: string, depID: string | null) {
  const qc = useQueryClient()
  const { connected } = useRealtime()
  const key = ['deploy-log', appID, depID] as const
  const query = useQuery({
    queryKey: key,
    enabled: !!depID,
    queryFn: () =>
      apiGet<DeploymentLog>(`/api/v1/apps/${appID}/deployments/${depID}/log`),
  })

  useRealtimeEvent((ev, replay) => {
    if (!depID || replay) return
    const payload =
      typeof ev.payload === 'string' ? parsePayload(ev.payload) : ev.payload
    const eventDeploymentID =
      ev.resource_id || (payload?.deployment_id as string | undefined)
    if (ev.type !== 'deploy.build.log' || eventDeploymentID !== depID) return
    const line = (payload?.line as string | undefined) || ev.message
    if (!line) return
    qc.setQueryData<DeploymentLog>(key, (old) => {
      const base = old ?? { number: 0, status: 'building', error: '', content: '' }
      return { ...base, content: base.content ? base.content + '\n' + line : line }
    })
  })

  const prevConnected = useRef(connected)
  useEffect(() => {
    if (prevConnected.current === false && connected === true && depID) {
      qc.invalidateQueries({ queryKey: ['deploy-log', appID, depID] })
    }
    prevConnected.current = connected
  }, [connected, depID, appID, qc])

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
