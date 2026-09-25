import { Info } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { IconButton } from './icon-button'
import { Tooltip } from './tooltip'

const meta = {
  component: Tooltip,
  title: 'Overlays/Tooltip',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Tooltip>
export default meta
type Story = StoryObj<typeof meta>
export const Help: Story = {
  args: {
    trigger: (
      <IconButton label="More information">
        <Info size={18} />
      </IconButton>
    ),
    children: 'The health check runs every thirty seconds.',
  },
  play: async ({ canvas }) => {
    await userEvent.hover(canvas.getByRole('button', { name: 'More information' }))
    await expect(canvas.getByRole('tooltip')).toBeVisible()
  },
}
