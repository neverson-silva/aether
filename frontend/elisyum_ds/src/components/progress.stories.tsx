import type { Meta, StoryObj } from '@storybook/react-vite'
import { Progress } from './progress'

const meta = {
  component: Progress,
  title: 'Components/Progress',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Progress>
export default meta
type Story = StoryObj<typeof meta>
export const Deploying: Story = {
  args: { label: 'Deployment progress', value: 64 },
  render: (args) => (
    <div className="w-96">
      <Progress {...args} />
    </div>
  ),
}
export const Empty: Story = {
  args: { label: 'Deployment progress', value: 0 },
  render: (args) => (
    <div className="w-96">
      <Progress {...args} />
    </div>
  ),
}
