import type { Meta, StoryObj } from '@storybook/react-vite'
import { Chart } from './chart'

const meta = {
  component: Chart,
  title: 'Data/Chart',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Chart>
export default meta
type Story = StoryObj<typeof meta>
export const Requests: Story = {
  args: {
    title: 'Requests per minute',
    labels: ['10:00', '10:15', '10:30', '10:45'],
    series: [{ label: 'Requests', values: [20, 42, 34, 68] }],
  },
  render: (args) => (
    <div className="w-[34rem]">
      <Chart {...args} />
    </div>
  ),
}
