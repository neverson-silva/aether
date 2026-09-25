import type { Meta, StoryObj } from '@storybook/react-vite'
import { NavigationMenu } from './navigation-menu'

const meta = {
  component: NavigationMenu,
  title: 'Navigation/NavigationMenu',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof NavigationMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Destinations: Story = {
  args: {
    items: [
      { label: 'Applications', href: '/applications', active: true },
      { label: 'Deployments', href: '/deployments' },
      { label: 'Activity', href: '/activity' },
    ],
  },
}
