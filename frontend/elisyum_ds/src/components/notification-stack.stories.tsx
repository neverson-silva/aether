import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { NotificationStack } from './notification-stack'

const meta = {
  component: NotificationStack,
  title: 'Feedback/NotificationStack',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof NotificationStack>
export default meta
type Story = StoryObj<typeof meta>
export const Notifications: Story = {
  args: {
    items: [
      {
        id: '1',
        title: 'Deployment complete',
        description: 'API service is healthy in production.',
        tone: 'success',
        onDismiss: fn(),
      },
      {
        id: '2',
        title: 'Action needed',
        description: 'One environment variable is missing.',
        tone: 'warning',
      },
    ],
  },
  play: async ({ args, canvas }) => {
    await expect(canvas.getByRole('region', { name: 'Notifications' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Dismiss notification' }))
    await expect(args.items[0].onDismiss).toHaveBeenCalled()
  },
}
