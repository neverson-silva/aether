import type { Meta, StoryObj } from '@storybook/react-vite'
import { Bell, MagnifyingGlass } from '@phosphor-icons/react'
import { HeaderAction } from './header-action'

const meta = {
  component: HeaderAction,
  title: 'Components/HeaderAction',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof HeaderAction>
export default meta
type Story = StoryObj<typeof meta>

export const Search: Story = {
  args: { label: 'Search resources', children: <MagnifyingGlass size={19} /> },
}

export const Notifications: Story = {
  args: { label: 'Open notifications', children: <Bell size={19} /> },
}
