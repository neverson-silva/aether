import { createFileRoute } from '@tanstack/react-router'
import { DatabaseField } from '../../../pages/database-field'

export const Route = createFileRoute('/_shell/databases/')({
  component: DatabaseField,
})
