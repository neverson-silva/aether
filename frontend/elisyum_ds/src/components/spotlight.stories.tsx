import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Spotlight } from './spotlight'

const meta = {
  component: Spotlight,
  title: 'Navigation/Spotlight',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Spotlight>
export default meta
type Story = StoryObj<typeof meta>
export const Announcement: Story = {
  args: {
    trigger: 'Open spotlight',
    title: 'A clearer deployment workflow',
    children: (
      <p className="text-body text-text-secondary">
        Review the deployment plan, health checks, and recovery actions before shipping.
      </p>
    ),
  },
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open spotlight' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(
      canvas.getByText(
        'Review the deployment plan, health checks, and recovery actions before shipping.',
      ),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
  },
}
