import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Popover } from './popover'

const meta = {
  component: Popover,
  title: 'Overlays/Popover',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Popover>
export default meta
type Story = StoryObj<typeof meta>
export const ResourceContext: Story = {
  args: {
    trigger: 'View details',
    title: 'Production',
    children: '12 services are healthy in this environment.',
  },
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'View details' }))
    await expect(canvas.getByRole('dialog')).toBeVisible()
    await expect(
      canvas.getByText('12 services are healthy in this environment.'),
    ).toBeVisible()
  },
}

export const AnchorOverlap: Story = {
  args: {
    children: 'Notifications content.',
    placement: 'anchor-overlap-center',
    trigger: 'Notifications',
  },
  play: async ({ canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Notifications' }))
    const trigger = canvas.getByRole('button', { name: 'Notifications' })
    const popup = canvas.getByRole('dialog')
    const triggerRect = trigger.getBoundingClientRect()
    const popupRect = popup.getBoundingClientRect()
    await expect(popupRect.top).toBeLessThan(triggerRect.bottom)
    const triggerCenter = triggerRect.left + triggerRect.width / 2
    const popupCenter = popupRect.left + popupRect.width / 2
    await expect(Math.abs(popupCenter - triggerCenter)).toBeLessThanOrEqual(8)
  },
}
