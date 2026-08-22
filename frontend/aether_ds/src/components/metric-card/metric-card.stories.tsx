import type { Meta, StoryObj } from '@storybook/react'
import { MetricCard } from './metric-card'

const meta = {
  title: 'Data/Metric Card',
  component: MetricCard,
  tags: ['autodocs'],
} satisfies Meta<typeof MetricCard>
export default meta
type Story = StoryObj<typeof meta>
export const Overview: Story = {
  args: {
    label: 'Successful deploys',
    value: '98.4',
    unit: '%',
    delta: '+4.2%',
    period: 'vs last week',
    trend: 'up',
    status: 'success',
  },
}
