import type { Meta, StoryObj } from '@storybook/react'
import { BulkActionBar } from './bulk-action-bar'

const meta = {
  title: 'Patterns/Bulk Action Bar',
  component: BulkActionBar,
  tags: ['autodocs'],
} satisfies Meta<typeof BulkActionBar>
export default meta
type Story = StoryObj<typeof meta>
export const Selected: Story = {
  args: {
    selectedCount: 4,
    actions: [
      { id: 'restart', label: 'Restart' },
      { id: 'delete', label: 'Delete', destructive: true },
    ],
  },
}
