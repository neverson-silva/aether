import { createFileRoute } from '@tanstack/react-router'
import { ServiceDetail } from '../../../pages/service-detail'

export const Route = createFileRoute('/_shell/services/$serviceId')({
  validateSearch: (
    search,
  ): { from?: 'project' | 'services'; projectId?: string; tab?: string } => ({
    from:
      search.from === 'project' || search.from === 'services' ? search.from : undefined,
    projectId: typeof search.projectId === 'string' ? search.projectId : undefined,
    tab: typeof search.tab === 'string' ? search.tab : undefined,
  }),
  component: ServiceDetail,
})
