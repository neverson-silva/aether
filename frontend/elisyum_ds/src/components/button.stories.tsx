import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'

const meta = {
  component: Button,
  title: 'Components/Button',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Button>
export default meta
type Story = StoryObj<typeof meta>
export const Actions: Story = {
  args: { children: 'Deploy application', onClick: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Deploy application' }))
    await expect(args.onClick).toHaveBeenCalled()
  },
}
export const States: Story = {
  args: { children: 'Actions' },
  render: () => (
    <div className="flex flex-wrap gap-3">
      <Button>Deploy</Button>
      <Button tone="neutral">Save draft</Button>
      <Button tone="danger">Delete</Button>
      <Button tone="ghost">Cancel</Button>
      <Button loading>Working</Button>
      <Button disabled>Unavailable</Button>
    </div>
  ),
}
