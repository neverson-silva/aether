import { Controller, useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ToggleGroup } from './toggle-group'

const meta = {
  component: ToggleGroup,
  title: 'Components/ToggleGroup',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ToggleGroup>
export default meta
type Story = StoryObj<typeof meta>
const items = [
  { value: 'overview', label: 'Overview' },
  { value: 'logs', label: 'Logs' },
  { value: 'metrics', label: 'Metrics' },
]
export const Default: Story = {
  args: { defaultValue: 'overview', items, onValueChange: fn() },
  play: async ({ args, canvas }) => {
    const logs = canvas.getByRole('button', { name: 'Logs' })
    await userEvent.click(logs)
    await expect(logs).toHaveAttribute('aria-pressed', 'true')
    await expect(args.onValueChange).toHaveBeenCalledWith('logs')
  },
}
export const ReactHookForm: Story = {
  args: { items },
  render: () => {
    const form = useForm<{ view: string }>({ defaultValues: { view: 'overview' } })
    return (
      <Controller
        control={form.control}
        name="view"
        render={({ field }) => (
          <ToggleGroup
            items={items}
            value={field.value}
            onValueChange={field.onChange}
          />
        )}
      />
    )
  },
}
