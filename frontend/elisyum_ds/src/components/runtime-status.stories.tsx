import type { Meta, StoryObj } from '@storybook/react-vite'
import { RuntimeStatus } from './runtime-status'

const meta = {
  component: RuntimeStatus,
  title: 'Feedback/RuntimeStatus',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof RuntimeStatus>
export default meta
type Story = StoryObj<typeof meta>
export const States: Story = {
  args: { status: 'healthy' },
  render: () => (
    <div className="flex flex-wrap gap-3">
      <RuntimeStatus status="healthy" />
      <RuntimeStatus status="deploying" />
      <RuntimeStatus status="degraded" />
      <RuntimeStatus status="failed" />
      <RuntimeStatus status="unknown" />
    </div>
  ),
}
