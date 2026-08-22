import type { Meta, StoryObj } from '@storybook/react'
import { Drawer } from './drawer'

const meta = {
  title: 'Overlay/Drawer',
  component: Drawer,
  tags: ['autodocs'],
} satisfies Meta<typeof Drawer>
export default meta
type Story = StoryObj<typeof meta>
export const Left: Story = {
  args: {
    title: 'Navigation',
    description: 'Choose an area of the platform.',
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Open drawer
      </button>
    ),
    children: (
      <div className="space-y-2">
        <a
          href="/overview"
          className="block rounded-md p-2 hover:bg-surface-container"
        >
          Overview
        </a>
        <a
          href="/deployments"
          className="block rounded-md p-2 hover:bg-surface-container"
        >
          Deployments
        </a>
      </div>
    ),
  },
}
