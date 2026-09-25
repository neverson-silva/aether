import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { UserMenu } from './user-menu'

const meta = {
  component: UserMenu,
  title: 'Navigation/UserMenu',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof UserMenu>
export default meta
type Story = StoryObj<typeof meta>
export const Account: Story = {
  args: {
    name: 'Ada Lovelace',
    email: 'ada@example.com',
    items: [
      { label: 'Preferences', onSelect: fn() },
      { label: 'Sign out', danger: true },
    ],
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: /Ada Lovelace/ }))
    await expect(canvas.getByRole('menu')).toBeVisible()
    await userEvent.click(canvas.getByRole('menuitem', { name: 'Preferences' }))
    await expect(args.items?.[0]?.onSelect).toHaveBeenCalled()
  },
}
