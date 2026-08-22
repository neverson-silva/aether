import type { Meta, StoryObj } from '@storybook/react'
import { ErrorBoundaryUI } from './error-boundary-ui'

const meta = {
  title: 'Feedback/Error Boundary UI',
  component: ErrorBoundaryUI,
  tags: ['autodocs'],
} satisfies Meta<typeof ErrorBoundaryUI>
export default meta
type Story = StoryObj<typeof meta>
export const Recoverable: Story = {
  args: {
    error: new Error('Failed to resolve deployment data'),
    reportId: 'err_01HZX',
    reset: () => undefined,
  },
}
