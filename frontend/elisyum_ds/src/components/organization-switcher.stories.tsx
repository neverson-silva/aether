import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { OrganizationSwitcher } from './organization-switcher'

const meta = {
  component: OrganizationSwitcher,
  title: 'Navigation/OrganizationSwitcher',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof OrganizationSwitcher>
export default meta
type Story = StoryObj<typeof meta>
export const Workspaces: Story = {
  args: {
    options: [
      { value: 'aether', label: 'Aether' },
      { value: 'northstar', label: 'Northstar' },
    ],
    value: 'aether',
    onValueChange: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Organization' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Northstar' }))
    await expect(args.onValueChange).toHaveBeenCalledWith('northstar')
  },
}
