import { createFileRoute } from '@tanstack/react-router'
import { OnboardingSurface } from '../../pages/onboarding'

export const Route = createFileRoute('/onboarding/')({
  component: OnboardingSurface,
})
