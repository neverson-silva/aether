import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Carousel } from './carousel'

const meta = {
  component: Carousel,
  title: 'Components/Carousel',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Carousel>
export default meta
type Story = StoryObj<typeof meta>
export const Previews: Story = {
  args: {
    items: [
      <div className="grid h-48 place-items-center text-body">Preview one</div>,
      <div className="grid h-48 place-items-center text-body">Preview two</div>,
      <div className="grid h-48 place-items-center text-body">Preview three</div>,
    ],
  },
  render: (args) => (
    <div className="w-96">
      <Carousel {...args} />
    </div>
  ),
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Next slide' }))
    await expect(canvas.getByText('Preview two')).toBeVisible()
    await expect(canvas.getByText('02 / 03')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Previous slide' }))
    await expect(canvas.getByText('Preview one')).toBeVisible()
  },
}
