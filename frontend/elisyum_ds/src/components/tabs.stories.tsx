import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Tabs } from './tabs'

const meta = {
  component: Tabs,
  title: 'Navigation/Tabs',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Tabs>
export default meta
type Story = StoryObj<typeof meta>
const items = [
  {
    value: 'overview',
    label: 'Overview',
    content: 'Resource summary and current lifecycle state.',
  },
  { value: 'logs', label: 'Logs', content: 'Recent deployment and runtime output.' },
  {
    value: 'settings',
    label: 'Settings',
    content: 'Configuration and environment values.',
  },
]
export const Views: Story = {
  args: { items, defaultValue: 'overview', onValueChange: fn() },
  render: (args) => (
    <div className="w-[28rem]">
      <Tabs {...args} />
    </div>
  ),
  play: async ({ canvas, args }) => {
    await expect(
      canvas.getByText('Resource summary and current lifecycle state.'),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('tab', { name: 'Logs' }))
    await expect(
      canvas.getByText('Recent deployment and runtime output.'),
    ).toBeVisible()
    await expect(args.onValueChange).toHaveBeenCalledWith('logs')
  },
}
