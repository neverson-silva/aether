import type { Meta, StoryObj } from '@storybook/react'
import { ProgressRing } from './progress-ring'

const meta = {
  title: 'Data/Progress Ring',
  component: ProgressRing,
  tags: ['autodocs'],
} satisfies Meta<typeof ProgressRing>
export default meta
type Story = StoryObj<typeof meta>
export const Statuses: Story = {
  render: () => (
    <div className="flex gap-6">
      <ProgressRing value={72} />
      <ProgressRing value={100} status="success" />
      <ProgressRing indeterminate status="warning" />
    </div>
  ),
}
