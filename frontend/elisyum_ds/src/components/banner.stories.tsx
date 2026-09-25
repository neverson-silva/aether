import type { Meta, StoryObj } from '@storybook/react-vite'
import { Banner } from './banner'

const meta = {
  component: Banner,
  title: 'Feedback/Banner',
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof Banner>
export default meta
type Story = StoryObj<typeof meta>
export const Maintenance: Story = {
  args: {
    title: 'Scheduled maintenance',
    children: 'Deployments will be paused for ten minutes at 02:00 UTC.',
    tone: 'info',
  },
}
