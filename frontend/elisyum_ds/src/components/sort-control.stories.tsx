import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { SortControl } from './sort-control'

const meta = {
  component: SortControl,
  title: 'Data/SortControl',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof SortControl>
export default meta
type Story = StoryObj<typeof meta>
export const Cycle: Story = {
  args: { label: 'Last updated', onDirectionChange: fn() },
  render: (args) => {
    const [direction, setDirection] = useState<null | 'asc' | 'desc'>(null)
    return (
      <div className="flex items-center gap-2 text-supporting text-text-secondary">
        Last updated
        <SortControl
          {...args}
          direction={direction}
          onDirectionChange={(nextDirection) => {
            args.onDirectionChange(nextDirection)
            setDirection(nextDirection)
          }}
        />
      </div>
    )
  },
  play: async ({ args, canvas }) => {
    const control = canvas.getByRole('button', { name: 'Sort by Last updated' })
    await userEvent.click(control)
    await expect(args.onDirectionChange).toHaveBeenCalledWith('asc')
    await userEvent.click(control)
    await expect(args.onDirectionChange).toHaveBeenCalledWith('desc')
    await userEvent.click(control)
    await expect(args.onDirectionChange).toHaveBeenCalledWith(null)
  },
}
