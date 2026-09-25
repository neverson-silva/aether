import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { ElisyumCollapsible } from './collapsible'

const meta = {
  component: ElisyumCollapsible,
  title: 'Navigation/Collapsible',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ElisyumCollapsible>
export default meta
type Story = StoryObj<typeof meta>
export const Details: Story = {
  args: {
    title: 'Advanced settings',
    children: 'These settings affect deployment behavior and runtime health checks.',
  },
  render: (args) => (
    <div className="w-96">
      <ElisyumCollapsible {...args} />
    </div>
  ),
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Advanced settings' }))
    await expect(
      canvas.getByText(
        'These settings affect deployment behavior and runtime health checks.',
      ),
    ).toBeVisible()
  },
}
