import type { Meta, StoryObj } from '@storybook/react'
import { DateTimePicker } from './date-time-picker'

const meta = {
  title: 'Forms/Date Time Picker',
  component: DateTimePicker,
  tags: ['autodocs'],
} satisfies Meta<typeof DateTimePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { label: 'Deployment window', timezone: 'UTC' },
}
export const WithSeconds: Story = {
  args: {
    label: 'Scheduled at',
    withSeconds: true,
    defaultDate: '2026-08-21',
    defaultTime: '14:30:00',
  },
}
export const Invalid: Story = {
  args: { label: 'Start time', error: 'Choose a valid deployment time.' },
}
