import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { DateTimePicker } from './date-time-picker'
import { Field } from './field'

const meta = {
  component: DateTimePicker,
  title: 'Forms/DateTimePicker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DateTimePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Schedule: Story = {
  args: { 'aria-label': 'Schedule deployment', onChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Schedule deployment' }))
    await userEvent.click(canvas.getByRole('button', { name: 'Today' }))
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Invalid: Story = {
  render: () => (
    <div className="w-96">
      <Field
        id="schedule"
        label="Schedule deployment"
        error="Choose a valid date and time."
      >
        <DateTimePicker id="schedule" />
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ scheduled: string }>()
    return (
      <DateTimePicker
        aria-label="Schedule deployment"
        {...form.register('scheduled')}
      />
    )
  },
}
