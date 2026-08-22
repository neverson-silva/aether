import type { Meta, StoryObj } from '@storybook/react'
import { Sheet } from './sheet'

const meta = {
  title: 'Overlay/Sheet',
  component: Sheet,
  tags: ['autodocs'],
} satisfies Meta<typeof Sheet>
export default meta
type Story = StoryObj<typeof meta>
export const Mobile: Story = {
  args: {
    title: 'Filters',
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Open sheet
      </button>
    ),
    children: (
      <div className="text-body-sm text-muted-foreground">
        Filter controls belong here.
      </div>
    ),
  },
}
