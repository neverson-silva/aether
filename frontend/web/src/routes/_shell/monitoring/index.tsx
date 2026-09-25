import { createFileRoute } from '@tanstack/react-router'
import { MonitoringField } from '../../../pages/monitoring-field'

export const Route = createFileRoute('/_shell/monitoring/')({
  component: MonitoringField,
})
