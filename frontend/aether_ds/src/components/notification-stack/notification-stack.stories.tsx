import type { Meta, StoryObj } from '@storybook/react'
import { NotificationStack } from './notification-stack'

const meta = {
  title: 'Feedback/Notification Stack',
  component: NotificationStack,
  tags: ['autodocs'],
} satisfies Meta<typeof NotificationStack>
export default meta
type Story = StoryObj<typeof meta>
export const Activity: Story = {
  args: {
    notifications: [
      {
        id: '1',
        title: 'Deployment completed',
        description: 'aether-api is serving traffic.',
        tone: 'success',
        unread: true,
        timestamp: '4 minutes ago',
      },
      {
        id: '2',
        title: 'Approval required',
        description: 'Production deploy is waiting for review.',
        tone: 'warning',
        timestamp: '12 minutes ago',
      },
    ],
  },
}
