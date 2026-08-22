import type { Meta, StoryObj } from '@storybook/react'
import { Message } from './message'

const meta = {
  title: 'Feedback/Message',
  component: Message,
  tags: ['autodocs'],
} satisfies Meta<typeof Message>
export default meta
type Story = StoryObj<typeof meta>
export const InfoMessage: Story = {
  args: {
    title: 'Deployment queued',
    children: 'The runner will start when capacity is available.',
  },
}
export const ErrorMessage: Story = {
  args: {
    tone: 'error',
    title: 'Deployment failed',
    children: 'Review the build output and try again.',
  },
}
