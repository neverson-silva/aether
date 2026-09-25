import { createFileRoute } from '@tanstack/react-router'
import { ProjectComposer } from '../../../pages/project-composer'

export const Route = createFileRoute('/_shell/projects/new')({
  component: ProjectComposer,
})
