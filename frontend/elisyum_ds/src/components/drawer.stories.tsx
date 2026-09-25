import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Drawer } from './drawer'

const meta = {
  component: Drawer,
  title: 'Overlays/Drawer',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Drawer>
export default meta
type Story = StoryObj<typeof meta>
export const Inspector: Story = {
  args: {
    trigger: 'Open inspector',
    title: 'Resource inspector',
    children: (
      <p className="text-body text-text-secondary">
        Inspect runtime status, domains, and recent events in context.
      </p>
    ),
  },
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open inspector' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(
      canvas.getByText(
        'Inspect runtime status, domains, and recent events in context.',
      ),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('dialog')).not.toBeInTheDocument()
  },
}
