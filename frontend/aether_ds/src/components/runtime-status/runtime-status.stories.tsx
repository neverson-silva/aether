import type { Meta, StoryObj } from '@storybook/react'
import { RuntimeStatus } from './runtime-status'

const meta = {
  title: 'Feedback/Runtime Status',
  component: RuntimeStatus,
  tags: ['autodocs'],
} satisfies Meta<typeof RuntimeStatus>
export default meta
type Story = StoryObj<typeof meta>
export const Statuses: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <RuntimeStatus status="healthy" />
      <RuntimeStatus status="deploying" live />
      <RuntimeStatus status="degraded" />
      <RuntimeStatus status="failed" />
    </div>
  ),
}
