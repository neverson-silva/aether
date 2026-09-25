import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { EnvironmentSwitcher } from './environment-switcher'

const meta = {
  component: EnvironmentSwitcher,
  title: 'Navigation/EnvironmentSwitcher',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof EnvironmentSwitcher>
export default meta
type Story = StoryObj<typeof meta>
export const Environments: Story = {
  args: {
    options: [
      { value: 'production', label: 'Production' },
      { value: 'staging', label: 'Staging' },
    ],
    value: 'production',
    onValueChange: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Environment' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Staging' }))
    await expect(args.onValueChange).toHaveBeenCalledWith('staging')
  },
}
