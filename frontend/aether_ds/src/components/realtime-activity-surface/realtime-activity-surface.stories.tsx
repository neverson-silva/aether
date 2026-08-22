import type { Meta, StoryObj } from '@storybook/react'
import { RealtimeActivitySurface } from './realtime-activity-surface'

const meta = {
  title: 'Data/Realtime Activity Surface',
  component: RealtimeActivitySurface,
  tags: ['autodocs'],
} satisfies Meta<typeof RealtimeActivitySurface>
export default meta
type Story = StoryObj<typeof meta>
export const Live: Story = {
  args: {
    connected: true,
    realtime: true,
    unreadCount: 3,
    items: [
      {
        id: '1',
        title: 'Deployment completed',
        timestamp: 'Now',
        type: 'success',
        unread: true,
      },
      {
        id: '2',
        title: 'Health check passed',
        timestamp: '1m ago',
        type: 'info',
      },
    ],
  },
}
