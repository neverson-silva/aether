import { createFileRoute } from '@tanstack/react-router'
import { EmptyWorkspace } from '../../../pages/empty-workspace'

export const Route = createFileRoute('/_shell/members/')({
  component: EmptyWorkspace,
})
