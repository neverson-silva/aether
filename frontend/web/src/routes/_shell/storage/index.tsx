import { createFileRoute } from '@tanstack/react-router'
import { StorageField } from '../../../pages/storage-field'

export const Route = createFileRoute('/_shell/storage/')({
  component: StorageField,
})
