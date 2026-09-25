import { AppWindow, Gear, Stack } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Sidebar } from './sidebar'

const meta = {
  component: Sidebar,
  title: 'Navigation/Sidebar',
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof Sidebar>
export default meta
type Story = StoryObj<typeof meta>
export const Workspace: Story = {
  args: {
    brand: <span className="font-medium">Aether</span>,
    items: [
      {
        label: 'Applications',
        href: '/applications',
        active: true,
        icon: <AppWindow size={18} />,
      },
      { label: 'Deployments', href: '/deployments', icon: <Stack size={18} /> },
      { label: 'Settings', href: '/settings', icon: <Gear size={18} /> },
    ],
  },
  play: async ({ canvas }) => {
    const primary = canvas.getByRole('navigation', { name: 'Primary' })
    await expect(primary).toBeVisible()
    await expect(canvas.getByRole('link', { name: 'Applications' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    await userEvent.click(canvas.getByRole('link', { name: 'Deployments' }))
  },
}
