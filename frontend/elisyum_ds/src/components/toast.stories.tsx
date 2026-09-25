import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from './button'
import { Toast, ToastProvider } from './toast'

const meta = {
  component: Toast,
  title: 'Feedback/Toast',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Toast>
export default meta
type Story = StoryObj<typeof meta>
export const Provider: Story = {
  args: { message: 'Deployment queued' },
  render: (args) => (
    <ToastProvider>
      <Toast {...args} />
    </ToastProvider>
  ),
}
