import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ApprovalFlow } from './approval-flow'

const meta = {
  component: ApprovalFlow,
  title: 'Patterns/ApprovalFlow',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ApprovalFlow>
export default meta
type Story = StoryObj<typeof meta>
export const Review: Story = {
  args: {
    people: [
      { id: '1', name: 'Ada Lovelace', role: 'Workspace owner' },
      { id: '2', name: 'Grace Hopper', role: 'Release reviewer', decision: 'approved' },
    ],
    onApprove: fn(),
    onReject: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Approve' }))
    await expect(args.onApprove).toHaveBeenCalledWith('1')
  },
}
