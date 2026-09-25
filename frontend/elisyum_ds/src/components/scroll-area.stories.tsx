import type { Meta, StoryObj } from '@storybook/react-vite'
import { ScrollArea } from './scroll-area'

const meta = {
  component: ScrollArea,
  title: 'Foundations/ScrollArea',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ScrollArea>
export default meta
type Story = StoryObj<typeof meta>
export const Logs: Story = {
  args: {
    children: (
      <div className="grid gap-2 p-3">
        {Array.from({ length: 12 }, (_, index) => (
          <div
            key={index}
            className="font-technical text-log text-text-secondary"
          >
            10:42:{index.toString().padStart(2, '0')} request completed
          </div>
        ))}
      </div>
    ),
    className: 'h-48 w-96 rounded-md border border-border-subtle bg-surface-1',
  },
}
