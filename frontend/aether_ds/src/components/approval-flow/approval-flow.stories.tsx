import type { Meta, StoryObj } from '@storybook/react'
import { ApprovalFlow } from './approval-flow'

const meta = {
  title: 'Patterns/Approval Flow',
  component: ApprovalFlow,
  tags: ['autodocs'],
} satisfies Meta<typeof ApprovalFlow>
export default meta
type Story = StoryObj<typeof meta>
export const Pending: Story = {
  args: {
    requester: 'Neverson Silva',
    policy: 'Production deployments require one approver.',
    approvers: [
      { id: '1', name: 'Ana Costa', role: 'Platform owner', status: 'pending' },
    ],
  },
}
