import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { OfflineIndicator } from './offline-indicator'

const meta = {
  component: OfflineIndicator,
  title: 'Feedback/OfflineIndicator',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof OfflineIndicator>
export default meta
type Story = StoryObj<typeof meta>
export const Offline: Story = {
  args: { state: 'offline' },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('alert')).toBeVisible()
    await expect(canvas.getByText('You are offline')).toBeVisible()
  },
}
export const Reconnecting: Story = {
  args: { state: 'reconnecting' },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('status')).toBeVisible()
    await expect(canvas.getByText('Reconnecting')).toBeVisible()
  },
}
