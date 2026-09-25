import { FolderOpen } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Button } from './button'
import { EmptyState } from './empty-state'

const meta = {
  component: EmptyState,
  title: 'Feedback/EmptyState',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof EmptyState>
export default meta
type Story = StoryObj<typeof meta>
export const NoResources: Story = {
  args: {
    title: 'No resources yet',
    description: 'Create your first application to start managing deployments.',
    icon: <FolderOpen size={28} />,
    action: <Button>Create application</Button>,
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByText('No resources yet')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Create application' }))
  },
}
