import { createFileRoute } from '@tanstack/react-router'
import { RegistryField } from '../../../pages/registry-field'

export const Route = createFileRoute('/_shell/registry/')({
  component: RegistryField,
})
