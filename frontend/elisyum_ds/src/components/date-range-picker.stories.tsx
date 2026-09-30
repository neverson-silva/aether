import type { Meta, StoryObj } from '@storybook/react-vite'
import { fn } from 'storybook/test'
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
    onChange: fn(),
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
