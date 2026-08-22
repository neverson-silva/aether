import type { Meta, StoryObj } from '@storybook/react'
import { AlertDialog } from './alert-dialog'

const meta = {
  title: 'Overlay/Alert Dialog',
  component: AlertDialog,
  tags: ['autodocs'],
} satisfies Meta<typeof AlertDialog>
export default meta
type Story = StoryObj<typeof meta>
export const Destructive: Story = {
  args: {
    title: 'Delete service?',
    description:
      'This permanently removes the service and its deployment history.',
    trigger: (
      <button
        type="button"
        className="rounded-md bg-destructive px-3 py-2 text-destructive-foreground"
      >
        Delete service
      </button>
    ),
    confirmLabel: 'Delete',
  },
}
