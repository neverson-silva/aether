import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPatch } from '../api/client'
import { qk } from './query-keys'

type DomainInput = {
  host: string
  https: boolean
  container_port: number
}

export function useUpdateDomain(kind: string, id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ domainID, body }: { domainID: string; body: DomainInput }) =>
      apiPatch(`/api/v1/${kind}/${id}/domains/${domainID}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(kind, id) }),
  })
}
