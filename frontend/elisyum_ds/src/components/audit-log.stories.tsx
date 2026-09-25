import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { AuditLog } from './audit-log'

const meta = {
  component: AuditLog,
  title: 'Data/AuditLog',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof AuditLog>
export default meta
type Story = StoryObj<typeof meta>
export const Events: Story = {
  args: {
    entries: [
      {
        id: '1',
        actor: 'Ada Lovelace',
        action: 'updated',
        target: 'API service',
        timestamp: '10:42',
      },
      {
        id: '2',
        actor: 'Grace Hopper',
        action: 'approved',
        target: 'Release v2.4',
        timestamp: '10:39',
      },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('region', { name: 'Audit log' })).toBeVisible()
    await expect(canvas.getByRole('columnheader', { name: 'Actor' })).toBeVisible()
    await expect(canvas.getByText('Release v2.4')).toBeVisible()
  },
}
