import { createFileRoute, useRouter } from '@tanstack/react-router'
import { FleetOverview } from '../../pages/fleet-overview'

export const Route = createFileRoute('/_shell/')({
  component: function FleetRoute() {
    const router = useRouter()
    return <FleetOverview onNavigate={(path) => router.navigate({ to: path })} />
  },
})
