import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ContextMenu } from './context-menu'

const meta = {
  component: ContextMenu,
  title: 'Navigation/ContextMenu',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ContextMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Resource: Story = {
  args: {
    children: (
      <div className="grid h-32 w-72 rounded-lg border border-dashed border-border-default text-supporting text-text-tertiary place-items-center">
        Right-click resource
      </div>
    ),
    items: [
      { label: 'Open resource', onSelect: fn() },
      { label: 'Copy identifier' },
      { label: 'Delete', danger: true },
    ],
  },
  play: async ({ args, canvas }) => {
    await userEvent.pointer({
      keys: '[MouseRight]',
      target: canvas.getByText('Right-click resource'),
    })
    await expect(canvas.getByRole('menu')).toBeVisible()
    await userEvent.click(canvas.getByRole('menuitem', { name: 'Open resource' }))
    await expect(args.items[0].onSelect).toHaveBeenCalled()
  },
}
