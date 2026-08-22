import type { Meta, StoryObj } from '@storybook/react'
import { MessageScroller } from './message-scroller'

const meta = {
  title: 'Feedback/Message Scroller',
  component: MessageScroller,
  tags: ['autodocs'],
} satisfies Meta<typeof MessageScroller>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    items: [
      {
        id: '1',
        title: 'Build passed',
        children: 'All checks completed.',
        tone: 'success',
      },
      {
        id: '2',
        title: 'Queue delay',
        children: 'Runner capacity is limited.',
        tone: 'warning',
      },
    ],
  },
}
