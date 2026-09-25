import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Sheet } from './sheet'

const meta = {
  component: Sheet,
  title: 'Overlays/Sheet',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Sheet>
export default meta
type Story = StoryObj<typeof meta>
export const CommandSurface: Story = {
  args: {
    trigger: 'Open command surface',
    title: 'Quick actions',
    children: (
      <p className="text-body text-text-secondary">
        Run a deployment, open logs, or change the current environment.
      </p>
    ),
    side: 'bottom',
  },
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open command surface' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(
      canvas.getByText(
        'Run a deployment, open logs, or change the current environment.',
      ),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('dialog')).not.toBeInTheDocument()
  },
}
