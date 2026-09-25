import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { WorkspaceSwitcher } from './workspace-switcher'

const meta = {
  component: WorkspaceSwitcher,
  title: 'Navigation/WorkspaceSwitcher',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof WorkspaceSwitcher>

export default meta
type Story = StoryObj<typeof meta>

export const Organizations: Story = {
  args: {
    options: [
      { id: 'aether', name: 'Aether', detail: 'Owner' },
      { id: 'northstar', name: 'Northstar', detail: 'Member' },
    ],
    value: 'aether',
    onValueChange: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: /Aether/ }))
    await expect(canvas.getByRole('menu')).toBeVisible()
    await userEvent.click(canvas.getByRole('menuitem', { name: /Northstar/ }))
    await expect(args.onValueChange).toHaveBeenCalledWith('northstar')
  },
}
