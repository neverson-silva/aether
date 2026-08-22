import type { Meta, StoryObj } from '@storybook/react'
import { ChartTooltipLegend } from './chart-tooltip-legend'

const meta = {
  title: 'Data/Chart Tooltip Legend',
  component: ChartTooltipLegend,
  tags: ['autodocs'],
} satisfies Meta<typeof ChartTooltipLegend>
export default meta
type Story = StoryObj<typeof meta>
export const Series: Story = {
  args: {
    title: 'Requests at 14:00',
    items: [
      { label: 'API', value: 1240, unit: 'req/s' },
      { label: 'Worker', value: 320, unit: 'jobs/s' },
    ],
  },
}
