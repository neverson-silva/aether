import type { Meta, StoryObj } from '@storybook/react'
import { Dialog } from './dialog'

const meta = {
  title: 'Overlay/Dialog',
  component: Dialog,
  tags: ['autodocs'],
} satisfies Meta<typeof Dialog>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    title: 'Deployment details',
    description: 'Review the current release before continuing.',
    trigger: (
      <button
        type="button"
        className="rounded-md bg-primary px-3 py-2 text-primary-foreground"
      >
        Open dialog
      </button>
    ),
    children: (
      <p className="text-body-sm text-muted-foreground">
        This action keeps your current environment context.
      </p>
    ),
  },
}
