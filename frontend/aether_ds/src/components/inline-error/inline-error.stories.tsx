import type { Meta, StoryObj } from '@storybook/react'
import { InlineError } from './inline-error'

const meta = {
  title: 'Feedback/Inline Error',
  component: InlineError,
  tags: ['autodocs'],
} satisfies Meta<typeof InlineError>
export default meta
type Story = StoryObj<typeof meta>
export const Recoverable: Story = {
  args: {
    title: 'Could not load deployments',
    message: 'The request timed out. Your existing data is still available.',
    requestId: 'req_01HZX',
    onRetry: () => undefined,
  },
}
