import type { Meta, StoryObj } from '@storybook/react'
import { CodeBlock } from './code-block'

const meta = {
  title: 'Components/Code Block',
  component: CodeBlock,
  tags: ['autodocs'],
} satisfies Meta<typeof CodeBlock>
export default meta
type Story = StoryObj<typeof meta>
export const Command: Story = {
  args: {
    title: 'Deploy command',
    language: 'bash',
    code: 'aether deploy --service api --environment production',
    collapsible: true,
  },
}
