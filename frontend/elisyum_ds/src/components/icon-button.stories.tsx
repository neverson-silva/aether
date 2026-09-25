import type { Meta, StoryObj } from '@storybook/react-vite'
import { Gear, X } from '@phosphor-icons/react'
import { IconButton } from './icon-button'

const meta = {
  component: IconButton,
  title: 'Components/IconButton',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof IconButton>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { label: 'Open settings', children: <Gear size={18} /> },
}
export const Close: Story = {
  args: { label: 'Close panel', children: <X size={18} />, size: 'sm' },
}
