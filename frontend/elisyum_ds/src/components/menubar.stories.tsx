import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Menubar } from './menubar'

const meta = {
  component: Menubar,
  title: 'Navigation/Menubar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Menubar>
export default meta
type Story = StoryObj<typeof meta>
export const Commands: Story = {
  args: {
    items: [{ label: 'File', onSelect: fn() }, { label: 'View' }, { label: 'Help' }],
  },
  play: async ({ args, canvas }) => {
    await expect(
      canvas.getByRole('menubar', { name: 'Application menu' }),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('menuitem', { name: 'File' }))
    await expect(args.items[0].onSelect).toHaveBeenCalled()
  },
}
