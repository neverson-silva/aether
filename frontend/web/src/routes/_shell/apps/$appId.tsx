import { createFileRoute } from '@tanstack/react-router'
import { AppDetail } from '../../../pages/app-detail'

export const Route = createFileRoute('/_shell/apps/$appId')({
  component: AppDetail,
})
