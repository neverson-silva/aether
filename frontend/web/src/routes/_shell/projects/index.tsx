import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ResourceInventory } from '../../../pages/resource-inventory'

export const Route = createFileRoute('/_shell/projects/')({
  component: function ProjectsRoute() {
    const router = useRouter()
    return (
      <ResourceInventory
        kind="projects"
        onNavigate={(path) => router.navigate({ to: path })}
      />
    )
  },
})
