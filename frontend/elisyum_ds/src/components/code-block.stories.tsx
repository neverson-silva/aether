import type { Meta, StoryObj } from '@storybook/react-vite'
import { CodeBlock } from './code-block'

const meta = {
  component: CodeBlock,
  title: 'Data/CodeBlock',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof CodeBlock>
export default meta
type Story = StoryObj<typeof meta>
export const Command: Story = {
  args: {
    language: 'shell',
    code: 'aether deploy --environment production\nDeployment queued.',
  },
  render: (args) => (
    <div className="w-[36rem]">
      <CodeBlock {...args} />
    </div>
  ),
}
