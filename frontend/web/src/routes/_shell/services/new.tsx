import { createFileRoute } from '@tanstack/react-router'
import { ServiceComposer } from '../../../pages/service-composer'

export const Route = createFileRoute('/_shell/services/new')({
  component: ServiceComposer,
})
