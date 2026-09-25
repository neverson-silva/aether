import { createFileRoute } from '@tanstack/react-router'
import { requireAuth } from '../../hooks/auth'
import { DatabaseStudio } from '../../pages/database-studio'

export const Route = createFileRoute('/studio/$databaseId')({
  beforeLoad: () => requireAuth(),
  validateSearch: (search): { returnTo?: string } => ({
    returnTo:
      typeof search.returnTo === 'string' && search.returnTo.startsWith('/services/')
        ? search.returnTo
        : undefined,
  }),
  component: StudioRoute,
})

function StudioRoute() {
  const { databaseId } = Route.useParams()
  const { returnTo } = Route.useSearch()
  return (
    <DatabaseStudio
      databaseId={databaseId}
      returnTo={returnTo ?? '/services'}
    />
  )
}
