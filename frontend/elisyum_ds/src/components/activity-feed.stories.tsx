import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { ActivityFeed } from './activity-feed'

const meta = {
  component: ActivityFeed,
  title: 'Data/ActivityFeed',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ActivityFeed>
export default meta
type Story = StoryObj<typeof meta>
export const Workspace: Story = {
  args: {
    items: [
      {
        id: '1',
        actor: 'Ada Lovelace',
        action: 'deployed API service',
        timestamp: '2 min ago',
      },
      {
        id: '2',
        actor: 'Grace Hopper',
        action: 'approved a release',
        timestamp: '12 min ago',
      },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('list', { name: 'Activity' })).toBeVisible()
    await expect(canvas.getAllByRole('listitem')).toHaveLength(2)
    await expect(canvas.getByText('deployed API service')).toBeVisible()
  },
}
