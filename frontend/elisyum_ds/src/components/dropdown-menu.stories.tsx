import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { DropdownMenu } from './dropdown-menu'

const meta = {
  component: DropdownMenu,
  title: 'Navigation/DropdownMenu',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DropdownMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Actions: Story = {
  args: {
    trigger: 'Actions',
    title: 'Service actions',
    items: [
      { label: 'Restart service', onSelect: fn() },
      { label: 'View logs' },
      { label: 'Delete service', danger: true },
    ],
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Actions' }))
    await expect(canvas.getByRole('menu')).toBeVisible()
    await userEvent.click(canvas.getByRole('menuitem', { name: 'Restart service' }))
    await expect(args.items[0].onSelect).toHaveBeenCalled()
  },
}
