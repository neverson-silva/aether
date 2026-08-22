import type { Meta, StoryObj } from '@storybook/react'
import { DropdownMenu } from './dropdown-menu'

const meta = {
  title: 'Navigation/Dropdown Menu',
  component: DropdownMenu,
  tags: ['autodocs'],
} satisfies Meta<typeof DropdownMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Actions
      </button>
    ),
    items: [
      { value: 'deploy', label: 'Deploy' },
      { value: 'rollback', label: 'Rollback' },
      { value: 'delete', label: 'Delete', destructive: true },
    ],
  },
}
