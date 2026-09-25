import { createFileRoute } from '@tanstack/react-router'
import { LoginSurface } from '../../pages/login'

export const Route = createFileRoute('/login/')({
  component: LoginSurface,
})
