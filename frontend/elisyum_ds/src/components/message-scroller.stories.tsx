import type { Meta, StoryObj } from '@storybook/react-vite'
import { MessageScroller } from './message-scroller'

const meta = {
  component: MessageScroller,
  title: 'Feedback/MessageScroller',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof MessageScroller>
export default meta
type Story = StoryObj<typeof meta>
export const Thread: Story = {
  args: {
    messages: [
      {
        id: '1',
        author: 'Ada Lovelace',
        children: 'The deployment is queued.',
        timestamp: '10:40',
      },
      {
        id: '2',
        author: 'Grace Hopper',
        children: 'I will review the output.',
        timestamp: '10:41',
        align: 'end',
      },
    ],
  },
  render: (args) => (
    <div className="h-72 w-[28rem] rounded-lg border border-border-subtle bg-surface-1">
      <MessageScroller {...args} />
    </div>
  ),
}
