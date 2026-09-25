import { Controller, useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Toggle } from './toggle'

const meta = {
  component: Toggle,
  title: 'Components/Toggle',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Toggle>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { children: 'Follow logs', onPressedChange: fn() },
  play: async ({ args, canvas }) => {
    const toggle = canvas.getByRole('button', { name: 'Follow logs' })
    await userEvent.click(toggle)
    await expect(toggle).toHaveAttribute('aria-pressed', 'true')
    await expect(args.onPressedChange).toHaveBeenCalledWith(true)
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ follow: boolean }>({ defaultValues: { follow: false } })
    return (
      <Controller
        control={form.control}
        name="follow"
        render={({ field }) => (
          <Toggle
            pressed={field.value}
            onPressedChange={field.onChange}
          >
            {field.value ? 'Following logs' : 'Follow logs'}
          </Toggle>
        )}
      />
    )
  },
}
