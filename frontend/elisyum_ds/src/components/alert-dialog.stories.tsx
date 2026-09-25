import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { AlertDialog } from './alert-dialog'

const meta = {
  component: AlertDialog,
  title: 'Overlays/AlertDialog',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof AlertDialog>
export default meta
type Story = StoryObj<typeof meta>
export const DeleteResource: Story = {
  args: {
    trigger: 'Delete service',
    title: 'Delete service?',
    description: 'The service and its deployment history will be permanently removed.',
    confirmLabel: 'Delete service',
    onConfirm: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Delete service' }))
    await expect(canvas.getByRole('alertdialog')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Delete service' }))
    await expect(args.onConfirm).toHaveBeenCalled()
  },
}

export const QuietTrigger: Story = {
  args: {
    trigger: 'Remove',
    triggerClassName: 'inline-flex min-h-9 cursor-pointer items-center justify-center rounded-lg px-3 text-label text-danger transition-colors hover:bg-danger-soft focus-visible:outline-2 focus-visible:outline-focus',
    title: 'Remove member?',
    description: 'This revokes the operator access boundary from the organization.',
    confirmLabel: 'Remove member',
    onConfirm: fn(),
  },
}
