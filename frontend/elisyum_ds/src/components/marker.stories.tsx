import type { Meta, StoryObj } from '@storybook/react-vite'
import { Marker } from './marker'

const meta = {
  component: Marker,
  title: 'Foundations/Marker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Marker>
export default meta
type Story = StoryObj<typeof meta>
export const Statuses: Story = {
  args: { label: 'Healthy', tone: 'success' },
  render: () => (
    <div className="flex gap-4">
      <Marker
        label="Queued"
        tone="warning"
      />
      <Marker
        label="Deploying"
        tone="accent"
      />
      <Marker
        label="Healthy"
        tone="success"
      />
      <Marker
        label="Failed"
        tone="danger"
      />
    </div>
  ),
}
