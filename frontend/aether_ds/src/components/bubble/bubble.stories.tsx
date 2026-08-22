import type { Meta, StoryObj } from '@storybook/react-vite'
import { Bubble } from './bubble'

const meta = {
  title: 'Components/Bubble',
  component: Bubble,
  tags: ['autodocs'],
} satisfies Meta<typeof Bubble>
export default meta
type Story = StoryObj<typeof meta>
export const Conversation: Story = {
  args: { children: 'Conversation' },
  render: () => (
    <div className="flex flex-col gap-3">
      <Bubble author="Aether Bot" timestamp="10:42" avatar="◉">
        Deployment started.
      </Bubble>
      <Bubble side="end" tone="primary" author="You">
        Show me the logs.
      </Bubble>
    </div>
  ),
}
