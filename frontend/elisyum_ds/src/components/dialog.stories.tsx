import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Dialog } from './dialog'

const meta = {
  component: Dialog,
  title: 'Overlays/Dialog',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Dialog>
export default meta
type Story = StoryObj<typeof meta>
export const EditResource: Story = {
  args: {
    onOpenChange: fn(),
    trigger: 'Edit resource',
    title: 'Edit service',
    description: 'Update the service name without changing its deployment.',
    children: (
      <input
        aria-label="Service name"
        className="min-h-10 w-full rounded-md border border-border-default bg-field px-3 text-body text-text-primary outline-none focus:border-focus focus:ring-2 focus:ring-focus/25"
        defaultValue="api-service"
      />
    ),
    footer: (
      <button
        className="inline-flex min-h-9 items-center rounded-md bg-action px-4 text-label text-on-action"
        type="button"
      >
        Save changes
      </button>
    ),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Edit resource' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(args.onOpenChange).toHaveBeenCalledWith(true)
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
    await expect(args.onOpenChange).toHaveBeenCalledWith(false)
  },
}
