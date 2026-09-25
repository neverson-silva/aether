import type { Meta, StoryObj } from '@storybook/react-vite'
import { Avatar } from './avatar'

const meta = {
  component: Avatar,
  title: 'Components/Avatar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Avatar>
export default meta
type Story = StoryObj<typeof meta>
export const Identity: Story = { args: { name: 'Ada Lovelace', alt: 'Ada Lovelace' } }
export const Sizes: Story = {
  args: { name: 'Aether Team' },
  render: () => (
    <div className="flex items-center gap-4">
      <Avatar
        name="Aether Team"
        size="sm"
      />
      <Avatar name="Aether Team" />
      <Avatar
        name="Aether Team"
        size="lg"
      />
    </div>
  ),
}
