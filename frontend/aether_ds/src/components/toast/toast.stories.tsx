import { Toast as BaseToast } from '@base-ui/react/toast'
import type { Meta, StoryObj } from '@storybook/react'
import { ToastProvider, useToast } from './toast'

const manager = BaseToast.createToastManager()
const meta = {
  title: 'Feedback/Toast',
  component: ToastProvider,
  tags: ['autodocs'],
} satisfies Meta<typeof ToastProvider>
export default meta
type Story = StoryObj<typeof meta>
function Trigger() {
  const toast = useToast()
  return (
    <button
      type="button"
      className="rounded-md border border-border px-3 py-2"
      onClick={() =>
        toast.add({
          title: 'Deployment queued',
          description: 'The release will start shortly.',
          tone: 'success',
        })
      }
    >
      Show toast
    </button>
  )
}
export const Interactive: Story = {
  args: { children: <Trigger /> },
  render: () => (
    <ToastProvider manager={manager}>
      <Trigger />
    </ToastProvider>
  ),
}
