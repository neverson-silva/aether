import { createFileRoute } from '@tanstack/react-router'
import { AppShell } from '../console'
import { requireAuth } from '../hooks/auth'

export const Route = createFileRoute('/_shell')({
  beforeLoad: () => requireAuth(),
  component: AppShell,
})
