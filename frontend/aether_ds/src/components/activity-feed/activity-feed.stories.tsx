import type { Meta, StoryObj } from '@storybook/react'
import { ActivityFeed } from './activity-feed'

const meta = {
  title: 'Patterns/Activity Feed',
  component: ActivityFeed,
  tags: ['autodocs'],
} satisfies Meta<typeof ActivityFeed>
export default meta
type Story = StoryObj<typeof meta>
export const Realtime: Story = {
  args: {
    realtime: true,
    items: [
      {
        id: '1',
        title: 'Deployment completed',
        description: 'aether-api is healthy.',
        timestamp: '2m ago',
        type: 'success',
        unread: true,
        actor: 'CI',
      },
      {
        id: '2',
        title: 'Variable updated',
        timestamp: '15m ago',
        type: 'change',
        actor: 'Neverson',
      },
    ],
  },
}
