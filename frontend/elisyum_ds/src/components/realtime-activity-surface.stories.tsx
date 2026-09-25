import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { RealtimeActivitySurface } from './realtime-activity-surface'

const meta = {
  component: RealtimeActivitySurface,
  title: 'Data/RealtimeActivitySurface',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof RealtimeActivitySurface>
export default meta
type Story = StoryObj<typeof meta>
export const Live: Story = {
  args: {
    items: [
      {
        id: '1',
        actor: 'Deploy worker',
        action: 'completed image build',
        timestamp: 'now',
      },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByText('Live')).toBeVisible()
    await expect(canvas.getByText('completed image build')).toBeVisible()
  },
}
export const Disconnected: Story = {
  args: {
    connected: false,
    stale: true,
    items: [
      {
        id: '1',
        actor: 'Deploy worker',
        action: 'last reported healthy',
        timestamp: '2 min ago',
      },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByText('Disconnected')).toBeVisible()
    await expect(canvas.getByText('last reported healthy')).toBeVisible()
  },
}
