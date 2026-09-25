import type { Meta, StoryObj } from '@storybook/react-vite'
import { Alert } from './alert'

const meta = {
  component: Alert,
  title: 'Feedback/Alert',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Alert>
export default meta
type Story = StoryObj<typeof meta>
export const Statuses: Story = {
  args: {
    title: 'Deployment queued',
    children: 'The deployment will start when the current build slot is available.',
  },
  render: () => (
    <div className="grid w-96 gap-3">
      <Alert
        title="Build complete"
        tone="success"
      >
        The image is ready for deployment.
      </Alert>
      <Alert
        title="Attention needed"
        tone="warning"
      >
        One environment variable is missing.
      </Alert>
      <Alert
        title="Deployment failed"
        tone="danger"
      >
        Review the build logs to retry.
      </Alert>
    </div>
  ),
}
