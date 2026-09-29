import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiPatch } from '../api/client'
import type { S3Destination } from '../api/types'

export function useUpdateS3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: unknown }) =>
      apiPatch<S3Destination>(`/api/v1/s3-destinations/${id}`, body),
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['s3'] }),
        qc.invalidateQueries({ queryKey: ['templates'] }),
        qc.invalidateQueries({ queryKey: ['templates-categories'] }),
      ]),
  })
}
