import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { BulkActionBar } from './bulk-action-bar'

const meta = {
  component: BulkActionBar,
  title: 'Patterns/BulkActionBar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof BulkActionBar>
export default meta
type Story = StoryObj<typeof meta>
export const Selected: Story = {
  args: {
    count: 3,
    actions: [
      { id: 'deploy', label: 'Deploy selected', onSelect: fn() },
      { id: 'delete', label: 'Delete', tone: 'danger' },
    ],
  },
  play: async ({ args, canvas }) => {
    await expect(canvas.getByRole('region', { name: 'Bulk actions' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Deploy selected' }))
    await expect(args.actions[0].onSelect).toHaveBeenCalled()
  },
}
