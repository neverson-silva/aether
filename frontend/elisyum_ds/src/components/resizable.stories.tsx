import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Resizable } from './resizable'

const meta = {
  component: Resizable,
  title: 'Patterns/Resizable',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Resizable>
export default meta
type Story = StoryObj<typeof meta>
export const SplitView: Story = {
  args: {
    first: <p className="text-body">Resource list</p>,
    second: <p className="text-body">Inspector</p>,
    onValueChange: fn(),
  },
  render: (args) => {
    const [value, setValue] = useState(50)
    return (
      <div className="w-[42rem]">
        <Resizable
          {...args}
          onValueChange={(nextValue) => {
            args.onValueChange?.(nextValue)
            setValue(nextValue)
          }}
          value={value}
        />
      </div>
    )
  },
  play: async ({ args, canvas }) => {
    const slider = canvas.getByRole('slider', { name: 'Panel split' })
    await userEvent.click(slider)
    await userEvent.keyboard('{ArrowRight}')
    await expect(canvas.getByText('51%')).toBeVisible()
    await expect(args.onValueChange).toHaveBeenCalledWith(51)
  },
}
