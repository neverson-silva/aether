import { useMutation } from '@tanstack/react-query'
import { apiPost } from '../api/client'
import type { DatabaseConnectionDetails } from '../api/types'

export function useRevealDatabaseConnection(dbId: string) {
  return useMutation({
    mutationFn: (scope: 'internal' | 'external') =>
      apiPost<DatabaseConnectionDetails>(
        `/api/v1/databases/${dbId}/connection/reveal`,
        { scope },
      ),
  })
}
