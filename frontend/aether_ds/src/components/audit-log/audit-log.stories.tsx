import type { Meta, StoryObj } from '@storybook/react'
import { AuditLog } from './audit-log'

const meta = {
  title: 'Data/Audit Log',
  component: AuditLog,
  tags: ['autodocs'],
} satisfies Meta<typeof AuditLog>
export default meta
type Story = StoryObj<typeof meta>
export const Events: Story = {
  args: {
    entries: [
      {
        id: '1',
        actor: 'Neverson',
        action: 'deployed',
        resource: 'aether-api',
        timestamp: 'Today, 12:40',
        requestId: 'req_01HZX',
        diff: 'image: v1.8.1 -> v1.8.2',
      },
      {
        id: '2',
        actor: 'CI',
        action: 'updated',
        resource: 'production',
        timestamp: 'Today, 11:12',
      },
    ],
  },
}
