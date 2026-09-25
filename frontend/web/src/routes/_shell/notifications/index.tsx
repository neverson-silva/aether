import { createFileRoute } from '@tanstack/react-router'
import { NotificationsField } from '../../../pages/notifications-field'

export const Route = createFileRoute('/_shell/notifications/')({
  component: NotificationsField,
})
