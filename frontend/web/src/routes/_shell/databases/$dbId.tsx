import { createFileRoute } from '@tanstack/react-router'
import { DatabaseDetail } from '../../../pages/database-detail'

export const Route = createFileRoute('/_shell/databases/$dbId')({
  component: DatabaseDetail,
})
