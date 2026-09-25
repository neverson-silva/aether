import { useMutation } from '@tanstack/react-query'
import { ApiError, getServer } from '../api/client'

export function useExportOrg() {
  return useMutation({
    mutationFn: async () => {
      const server = getServer()
      const res = await fetch(server + '/api/v1/org/export', { credentials: 'include' })
      if (!res.ok) throw new ApiError(res.status, 'export falhou')
      return res.text()
    },
  })
}
