import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Timeline } from './timeline'

const meta = {
  component: Timeline,
  title: 'Data/Timeline',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Timeline>
export default meta
type Story = StoryObj<typeof meta>
export const Deployment: Story = {
  args: {
    items: [
      {
        id: 'queued',
        title: 'Deployment queued',
        description: 'Waiting for a build slot.',
        timestamp: '10:42:12',
        status: 'Queued',
      },
      {
        id: 'build',
        title: 'Build complete',
        description: 'Image published successfully.',
        timestamp: '10:43:02',
        status: 'Succeeded',
        tone: 'success',
      },
      {
        id: 'live',
        title: 'Runtime healthy',
        timestamp: '10:44:18',
        status: 'Healthy',
        tone: 'success',
      },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('list', { name: 'Timeline' })).toBeVisible()
    await expect(canvas.getAllByRole('listitem')).toHaveLength(3)
    await expect(canvas.getByText('Runtime healthy')).toBeVisible()
  },
}
