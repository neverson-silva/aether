import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../api/client'

export function useAppCompose(composeID: string, enabled = true) {
  return useQuery({
    queryKey: ['compose-definition', composeID],
    enabled: !!composeID && enabled,
    queryFn: () => apiGet<{ compose: string }>(`/api/v1/compose/${composeID}`),
  })
}
