import { createFileRoute } from '@tanstack/react-router'
import { ProjectDetail } from '../../../pages/project-detail'

export const Route = createFileRoute('/_shell/projects/$projectId')({
  component: ProjectDetail,
})
