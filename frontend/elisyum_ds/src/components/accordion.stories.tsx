import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Accordion } from './accordion'

const meta = {
  component: Accordion,
  title: 'Navigation/Accordion',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Accordion>
export default meta
type Story = StoryObj<typeof meta>
const items = [
  {
    value: 'runtime',
    title: 'Runtime status',
    content: 'The service is running in the production environment.',
  },
  {
    value: 'network',
    title: 'Network details',
    content: 'Traffic is routed through the managed ingress.',
  },
]
export const Sections: Story = {
  args: { items, onValueChange: fn() },
  render: (args) => (
    <div className="w-96">
      <Accordion {...args} />
    </div>
  ),
  play: async ({ canvas, args }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Runtime status' }))
    await expect(
      canvas.getByText('The service is running in the production environment.'),
    ).toBeVisible()
    await expect(args.onValueChange).toHaveBeenCalled()
  },
}
