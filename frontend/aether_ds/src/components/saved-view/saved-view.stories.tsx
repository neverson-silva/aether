import type { Meta, StoryObj } from '@storybook/react'
import { SavedView } from './saved-view'

const meta = {
  title: 'Data/Saved View',
  component: SavedView,
  tags: ['autodocs'],
} satisfies Meta<typeof SavedView>
export default meta
type Story = StoryObj<typeof meta>
export const Views: Story = {
  args: {
    value: 'production',
    views: [
      {
        id: 'production',
        name: 'Production services',
        owner: 'You',
        favorite: true,
      },
      { id: 'errors', name: 'Recent errors', shared: true },
    ],
  },
}
