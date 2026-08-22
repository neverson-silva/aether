import type { Meta, StoryObj } from '@storybook/react'
import { VirtualizedList } from './virtualized-list'

const meta = {
  title: 'Data/Virtualized List',
  component: VirtualizedList,
  tags: ['autodocs'],
} satisfies Meta<typeof VirtualizedList>
export default meta
type Story = StoryObj<typeof meta>
export const ThousandsOfResources: Story = {
  args: { items: [], rowHeight: 40, renderItem: () => null },
  render: () => (
    <VirtualizedList
      items={Array.from(
        { length: 1000 },
        (_, index) => `Resource ${index + 1}`,
      )}
      rowHeight={40}
      renderItem={(item) => (
        <div className="border-b border-border px-3 py-2 text-body-sm">
          {item}
        </div>
      )}
    />
  ),
}
