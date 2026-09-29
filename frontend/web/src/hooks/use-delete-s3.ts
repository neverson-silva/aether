import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiDelete } from '../api/client'

export function useDeleteS3() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/s3-destinations/${id}`),
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['s3'] }),
        qc.invalidateQueries({ queryKey: ['templates'] }),
        qc.invalidateQueries({ queryKey: ['templates-categories'] }),
      ]),
  })
}
