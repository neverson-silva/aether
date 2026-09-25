import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ErrorBoundaryUI } from './error-boundary-ui'

const meta = {
  component: ErrorBoundaryUI,
  title: 'Feedback/ErrorBoundaryUI',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ErrorBoundaryUI>
export default meta
type Story = StoryObj<typeof meta>
export const Recoverable: Story = {
  args: {
    title: 'Runtime panel unavailable',
    description: 'The panel failed to load. Retry when the connection is stable.',
    onRetry: fn(),
  },
  play: async ({ args, canvas }) => {
    await expect(canvas.getByRole('alert')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Try again' }))
    await expect(args.onRetry).toHaveBeenCalled()
  },
}
