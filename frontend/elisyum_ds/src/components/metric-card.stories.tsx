import type { Meta, StoryObj } from '@storybook/react-vite'
import { MetricCard } from './metric-card'

const meta = {
  component: MetricCard,
  title: 'Data/MetricCard',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof MetricCard>
export default meta
type Story = StoryObj<typeof meta>
export const Deployments: Story = {
  args: {
    label: 'Successful deployments',
    value: '98.4%',
    status: '+4.2%',
    tone: 'success',
    trend: 'Compared with the previous 30 days.',
  },
  render: (args) => (
    <div className="w-72">
      <MetricCard {...args} />
    </div>
  ),
}
