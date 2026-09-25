import type { Meta, StoryObj } from '@storybook/react-vite'
import { Message } from './message'

const meta = {
  component: Message,
  title: 'Feedback/Message',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Message>
export default meta
type Story = StoryObj<typeof meta>
export const Conversation: Story = {
  args: {
    author: 'Ada Lovelace',
    children: 'The health check is passing in production.',
    timestamp: '10:42',
  },
}
