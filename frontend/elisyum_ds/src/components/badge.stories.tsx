import type { Meta, StoryObj } from '@storybook/react-vite'
import { Badge } from './badge'

const meta = {
  component: Badge,
  title: 'Components/Badge',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Badge>
export default meta
type Story = StoryObj<typeof meta>
export const Statuses: Story = {
  args: { children: 'Statuses' },
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge>Queued</Badge>
      <Badge tone="accent">Deploying</Badge>
      <Badge tone="success">Healthy</Badge>
      <Badge tone="warning">Degraded</Badge>
      <Badge tone="danger">Failed</Badge>
    </div>
  ),
}
