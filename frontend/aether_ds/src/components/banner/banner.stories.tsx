import type { Meta, StoryObj } from '@storybook/react'
import { Banner } from './banner'

const meta = {
  title: 'Feedback/Banner',
  component: Banner,
  tags: ['autodocs'],
} satisfies Meta<typeof Banner>
export default meta
type Story = StoryObj<typeof meta>
export const Maintenance: Story = {
  args: {
    title: 'Scheduled maintenance',
    description: 'The platform will be read-only from 02:00 to 02:30 UTC.',
    tone: 'warning',
    dismissible: true,
    action: (
      <button type="button" className="text-body-sm font-semibold text-primary">
        View status page
      </button>
    ),
  },
}
