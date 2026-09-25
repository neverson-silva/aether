import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { SavedView } from './saved-view'

const meta = {
  component: SavedView,
  title: 'Patterns/SavedView',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof SavedView>
export default meta
type Story = StoryObj<typeof meta>
export const Filters: Story = {
  args: {
    views: [
      { id: 'healthy', label: 'Healthy services' },
      { id: 'recent', label: 'Recently deployed' },
    ],
    value: 'healthy',
    onValueChange: fn(),
    onSave: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Saved view' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Recently deployed' }))
    await expect(args.onValueChange).toHaveBeenCalledWith('recent')
    await userEvent.click(canvas.getByRole('button', { name: 'Save current view' }))
    await expect(args.onSave).toHaveBeenCalled()
  },
}
