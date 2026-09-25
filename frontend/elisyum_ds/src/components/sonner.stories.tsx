import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from './button'
import { showToast, Sonner } from './sonner'

const meta = {
  component: Sonner,
  title: 'Feedback/Sonner',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Sonner>
export default meta
type Story = StoryObj<typeof meta>
export const Notifications: Story = {
  render: () => (
    <>
      <Sonner />
      <Button onClick={() => showToast('Deployment queued')}>Show toast</Button>
    </>
  ),
}
