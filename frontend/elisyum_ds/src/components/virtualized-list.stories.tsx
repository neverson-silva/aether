import type { Meta, StoryObj } from '@storybook/react-vite'
import { VirtualizedList } from './virtualized-list'

const meta = {
  component: VirtualizedList,
  title: 'Data/VirtualizedList',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof VirtualizedList<string>>
export default meta
type Story = StoryObj<typeof meta>
export const LongLog: Story = {
  args: {
    height: 240,
    items: Array.from({ length: 100 }, (_, index) => `Event ${index + 1}`),
    renderItem: (item) => (
      <div className="flex items-center border-b border-border-subtle px-3 text-supporting text-text-secondary">
        {String(item)}
      </div>
    ),
  },
}
