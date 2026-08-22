import type { Meta, StoryObj } from '@storybook/react'
import { OfflineIndicator } from './offline-indicator'

const meta = {
  title: 'Feedback/Offline Indicator',
  component: OfflineIndicator,
  tags: ['autodocs'],
} satisfies Meta<typeof OfflineIndicator>
export default meta
type Story = StoryObj<typeof meta>
export const States: Story = {
  args: { state: 'offline' },
  render: () => (
    <div className="flex flex-wrap gap-5">
      <OfflineIndicator state="offline" />
      <OfflineIndicator state="reconnecting" />
      <OfflineIndicator state="stale" />
      <OfflineIndicator state="queued" queuedCount={3} />
      <OfflineIndicator state="synced" />
    </div>
  ),
}
