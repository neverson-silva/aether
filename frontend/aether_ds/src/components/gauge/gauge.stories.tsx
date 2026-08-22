import type { Meta, StoryObj } from '@storybook/react'
import { Gauge } from './gauge'

const meta = {
  title: 'Data/Gauge',
  component: Gauge,
  tags: ['autodocs'],
} satisfies Meta<typeof Gauge>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { value: 72, label: '72%', status: 'success' },
}
