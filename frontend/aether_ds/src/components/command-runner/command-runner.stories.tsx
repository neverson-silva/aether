import type { Meta, StoryObj } from '@storybook/react'
import { CommandRunner } from './command-runner'

const meta = {
  title: 'Patterns/Command Runner',
  component: CommandRunner,
  tags: ['autodocs'],
} satisfies Meta<typeof CommandRunner>
export default meta
type Story = StoryObj<typeof meta>
export const Runtime: Story = {
  args: {
    command: 'aether status --service api',
    target: 'production',
    output: 'Service aether-api is healthy.',
    status: 'success',
  },
}
