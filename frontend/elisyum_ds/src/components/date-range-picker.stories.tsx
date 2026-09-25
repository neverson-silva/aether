import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { DateRangePicker } from './date-range-picker'
import { Field } from './field'

const meta = {
  component: DateRangePicker,
  title: 'Forms/DateRangePicker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DateRangePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Window: Story = {
  args: {
    startProps: { 'aria-label': 'Start date', onChange: fn() },
    endProps: { 'aria-label': 'End date', onChange: fn() },
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Start date' }))
    await userEvent.click(canvas.getByRole('button', { name: 'Today' }))
    await userEvent.click(canvas.getByRole('button', { name: 'End date' }))
    await userEvent.click(canvas.getByRole('button', { name: 'Today' }))
    await expect(args.startProps?.onChange).toHaveBeenCalled()
    await expect(args.endProps?.onChange).toHaveBeenCalled()
  },
  render: (args) => <DateRangePicker {...args} />,
}
export const Invalid: Story = {
  render: () => (
    <Field
      id="date-window"
      label="Release window"
      error="Select a valid release window."
    >
      <DateRangePicker id="date-window" />
    </Field>
  ),
}
