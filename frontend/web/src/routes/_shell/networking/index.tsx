import { createFileRoute } from '@tanstack/react-router'
import { NetworkingField } from '../../../pages/networking-field'

export const Route = createFileRoute('/_shell/networking/')({
  component: NetworkingField,
})
