import type { Meta, StoryObj } from '@storybook/react-vite'
import { ChartTooltipLegend } from './chart-tooltip-legend'

const meta = {
  component: ChartTooltipLegend,
  title: 'Data/ChartTooltipLegend',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ChartTooltipLegend>
export default meta
type Story = StoryObj<typeof meta>
export const Series: Story = {
  args: {
    items: [
      { label: 'Requests', value: '1,284', color: 'var(--ely-color-action)' },
      { label: 'Errors', value: '12', color: 'var(--ely-color-danger)' },
    ],
  },
}
