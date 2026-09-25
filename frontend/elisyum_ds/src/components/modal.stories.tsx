import type { Meta, StoryObj } from '@storybook/react-vite'
import { Modal } from './modal'

const meta = {
  component: Modal,
  title: 'Overlays/Modal',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Modal>
export default meta
type Story = StoryObj<typeof meta>
export const Information: Story = {
  args: {
    trigger: 'Open modal',
    title: 'Deployment details',
    children: (
      <p className="text-body text-text-secondary">
        The deployment is waiting for approval from the workspace owner.
      </p>
    ),
  },
}
