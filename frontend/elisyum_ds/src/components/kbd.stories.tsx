import type { Meta, StoryObj } from '@storybook/react-vite'
import { Kbd } from './kbd'

const meta = {
  component: Kbd,
  title: 'Foundations/Kbd',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Kbd>
export default meta
type Story = StoryObj<typeof meta>
export const Shortcut: Story = { args: { children: '⌘ K' } }
