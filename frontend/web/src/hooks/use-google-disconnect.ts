import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPost } from '../api/client'

export function useGoogleDisconnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiPost(`/api/v1/s3-destinations/${id}/google/disconnect`),
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['s3'] }),
        qc.invalidateQueries({ queryKey: ['templates'] }),
        qc.invalidateQueries({ queryKey: ['templates-categories'] }),
      ]),
  })
}
