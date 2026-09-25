import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ResourceInventory } from '../../../pages/resource-inventory'

export const Route = createFileRoute('/_shell/apps/')({
  component: function ApplicationsRoute() {
    const router = useRouter()
    return (
      <ResourceInventory
        kind="services"
        resourcePath="/apps"
        onNavigate={(path) => router.navigate({ to: path })}
      />
    )
  },
})
