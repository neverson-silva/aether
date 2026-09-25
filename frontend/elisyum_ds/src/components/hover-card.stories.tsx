import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { HoverCard } from './hover-card'

const meta = {
  component: HoverCard,
  title: 'Overlays/HoverCard',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof HoverCard>
export default meta
type Story = StoryObj<typeof meta>
export const ResourcePreview: Story = {
  args: {
    trigger: 'API service',
    title: 'API service',
    children: 'Production runtime with three healthy instances.',
  },
  play: async ({ canvas }) => {
    await userEvent.hover(canvas.getByRole('button', { name: 'API service' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(
      canvas.getByText('Production runtime with three healthy instances.'),
    ).toBeVisible()
  },
}
