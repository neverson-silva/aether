import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPost } from '../api/client'
import type { App } from '../api/types'
import { qk } from './query-keys'

type AppCreatePayload = Partial<App> & {
  root_folder?: string
  dist_folder?: string
  watch_paths?: string
  upload_id?: string
  install_command?: string
  build_command?: string
  start_command?: string
  env?: Array<{ name: string; value: string; secret: boolean }>
}

export function useCreateApp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { projectID: string; payload: AppCreatePayload }) =>
      apiPost<App>(`/api/v1/projects/${body.projectID}/apps`, body.payload),
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: qk.apps }),
        qc.invalidateQueries({ queryKey: qk.services }),
      ]),
  })
}
