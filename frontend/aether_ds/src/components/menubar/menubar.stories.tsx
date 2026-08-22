import type { Meta, StoryObj } from '@storybook/react'
import { Menubar } from './menubar'

const meta = {
  title: 'Navigation/Menubar',
  component: Menubar,
  tags: ['autodocs'],
} satisfies Meta<typeof Menubar>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    items: [
      {
        label: 'Project',
        items: [
          { value: 'settings', label: 'Settings' },
          { value: 'members', label: 'Members' },
        ],
      },
      {
        label: 'Deploy',
        items: [
          { value: 'new', label: 'New deployment' },
          { value: 'history', label: 'History' },
        ],
      },
    ],
  },
}
