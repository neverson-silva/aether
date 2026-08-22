import type { Meta, StoryObj } from '@storybook/react'
import { Spotlight } from './spotlight'

const meta = {
  title: 'Navigation/Spotlight',
  component: Spotlight,
  tags: ['autodocs'],
} satisfies Meta<typeof Spotlight>
export default meta
type Story = StoryObj<typeof meta>
export const GlobalSearch: Story = {
  args: {
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Open global search
      </button>
    ),
    items: [
      {
        id: 'service',
        label: 'Search services',
        description: 'Find a service or environment',
      },
      { id: 'deploy', label: 'New deployment', shortcut: 'D' },
    ],
  },
}
