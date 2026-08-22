import type { Meta, StoryObj } from '@storybook/react'
import { Popover } from './popover'

const meta = {
  title: 'Overlay/Popover',
  component: Popover,
  tags: ['autodocs'],
} satisfies Meta<typeof Popover>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    title: 'Environment details',
    description: 'Current deployment context.',
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Open details
      </button>
    ),
    children: <p className="text-body-sm">Production is protected.</p>,
  },
}
