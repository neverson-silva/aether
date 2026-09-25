import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ResourceInventory } from '../../../pages/resource-inventory'

export const Route = createFileRoute('/_shell/services/')({
  component: function ServicesRoute() {
    const router = useRouter()
    return (
      <ResourceInventory
        kind="services"
        onNavigate={(path) => router.navigate({ to: path })}
      />
    )
  },
})
