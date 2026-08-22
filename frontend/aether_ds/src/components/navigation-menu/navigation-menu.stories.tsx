import type { Meta, StoryObj } from '@storybook/react'
import { NavigationMenu } from './navigation-menu'

const meta = {
  title: 'Navigation/Navigation Menu',
  component: NavigationMenu,
  tags: ['autodocs'],
} satisfies Meta<typeof NavigationMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    items: [
      { label: 'Overview', href: '/overview' },
      {
        label: 'Resources',
        children: (
          <div className="grid gap-2">
            <a
              href="/services"
              className="rounded-md p-2 hover:bg-surface-container"
            >
              Services
            </a>
            <a
              href="/environments"
              className="rounded-md p-2 hover:bg-surface-container"
            >
              Environments
            </a>
          </div>
        ),
      },
    ],
  },
}
