import type { Meta, StoryObj } from '@storybook/react'
import { Timeline } from './timeline'

const meta = {
  title: 'Data/Timeline',
  component: Timeline,
  tags: ['autodocs'],
} satisfies Meta<typeof Timeline>
export default meta
type Story = StoryObj<typeof meta>
export const Deployment: Story = {
  args: {
    realtime: true,
    events: [
      {
        id: '1',
        title: 'Deployment completed',
        description: 'Version v1.8.2 is serving traffic.',
        timestamp: '4 min ago',
        status: 'complete',
        actor: 'Neverson',
      },
      {
        id: '2',
        title: 'Health checks running',
        timestamp: 'Now',
        status: 'active',
      },
    ],
  },
}
