import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { TimePicker } from './time-picker'

const meta = {
  component: TimePicker,
  title: 'Forms/TimePicker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof TimePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Window: Story = { args: { 'aria-label': 'Maintenance time' } }
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ time: string }>()
    return (
      <TimePicker
        aria-label="Maintenance time"
        {...form.register('time')}
      />
    )
  },
}
