import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Switch } from './switch'

const meta = {
  component: Switch,
  title: 'Components/Switch',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Switch>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { 'aria-label': 'Enable realtime updates', onChange: fn() },
  play: async ({ args, canvas }) => {
    const control = canvas.getByRole('checkbox', { name: 'Enable realtime updates' })
    await userEvent.click(control)
    await expect(control).toBeChecked()
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Checked: Story = {
  args: { 'aria-label': 'Enable automatic deployments', defaultChecked: true },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ realtime: boolean }>({ defaultValues: { realtime: true } })
    return (
      <label className="flex items-center gap-3 text-body">
        <Switch {...form.register('realtime')} />
        Enable realtime updates
      </label>
    )
  },
}
