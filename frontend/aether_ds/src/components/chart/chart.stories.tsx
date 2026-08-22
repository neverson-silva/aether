import type { Meta, StoryObj } from '@storybook/react'
import { Chart } from './chart'

const meta = {
  title: 'Data/Chart',
  component: Chart,
  tags: ['autodocs'],
} satisfies Meta<typeof Chart>
export default meta
type Story = StoryObj<typeof meta>
export const Line: Story = {
  args: {
    labels: ['10:00', '12:00', '14:00', '16:00'],
    series: [{ id: 'requests', label: 'Requests', values: [24, 42, 36, 64] }],
  },
}
