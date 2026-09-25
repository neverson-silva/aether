import { createFileRoute } from '@tanstack/react-router'
import { TraefikField } from '../../../pages/traefik-field'

export const Route = createFileRoute('/_shell/traefik/')({
  component: TraefikField,
})
