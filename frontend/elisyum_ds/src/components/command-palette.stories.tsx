import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { CommandPalette } from './command-palette'

const meta = {
  component: CommandPalette,
  title: 'Navigation/CommandPalette',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof CommandPalette>
export default meta
type Story = StoryObj<typeof meta>
export const Commands: Story = {
  args: {
    items: [
      { id: 'deploy', label: 'Deploy current application', keywords: ['release'] },
      {
        id: 'logs',
        label: 'Open runtime logs',
        keywords: ['observability'],
        onSelect: fn(),
      },
      { id: 'settings', label: 'Open settings' },
    ],
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Search commands' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await userEvent.type(
      canvas.getByRole('textbox', { name: 'Search commands' }),
      'observability',
    )
    await expect(
      canvas.getByRole('button', { name: 'Open runtime logs' }),
    ).toBeVisible()
    await expect(
      canvas.getByRole('button', { name: 'Open runtime logs' }),
    ).toHaveAttribute('aria-current', 'true')
    await userEvent.keyboard('{ArrowDown}{Enter}')
    await expect(args.items[1].onSelect).toHaveBeenCalled()
  },
}
