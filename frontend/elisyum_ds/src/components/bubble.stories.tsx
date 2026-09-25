import type { Meta, StoryObj } from '@storybook/react-vite'
import { Bubble } from './bubble'

const meta = {
  component: Bubble,
  title: 'Feedback/Bubble',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Bubble>
export default meta
type Story = StoryObj<typeof meta>
export const Conversation: Story = {
  args: { children: 'The deployment is ready for review.' },
  render: () => (
    <div className="grid w-96 gap-2">
      <Bubble>Can we review the deployment plan?</Bubble>
      <Bubble
        align="end"
        tone="accent"
      >
        The plan is ready for review.
      </Bubble>
    </div>
  ),
}
