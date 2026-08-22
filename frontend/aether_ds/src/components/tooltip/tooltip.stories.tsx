import { Info } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react'
import { Tooltip } from './tooltip'

const meta = {
  title: 'Overlay/Tooltip',
  component: Tooltip,
  tags: ['autodocs'],
} satisfies Meta<typeof Tooltip>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    content: 'Deployment status and details',
    children: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        <Info size={18} />
      </button>
    ),
  },
}
