import type { Meta, StoryObj } from '@storybook/react'
import { ContextMenu } from './context-menu'

const meta = {
  title: 'Navigation/Context Menu',
  component: ContextMenu,
  tags: ['autodocs'],
} satisfies Meta<typeof ContextMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Resource: Story = {
  args: {
    items: [
      { value: 'open', label: 'Open resource' },
      { value: 'copy', label: 'Copy ID' },
      { value: 'delete', label: 'Delete', destructive: true },
    ],
    children: (
      <div className="rounded-lg border border-dashed border-border p-12 text-center text-body-sm text-muted-foreground">
        Right click this resource
      </div>
    ),
  },
}
