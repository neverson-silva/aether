import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Table } from './table'

const meta = {
  component: Table,
  title: 'Data/Table',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Table>
export default meta
type Story = StoryObj<typeof meta>
export const Resources: Story = {
  args: {
    headers: ['Resource', 'Status', 'Updated'],
    rows: [
      ['API service', 'Healthy', '2 min ago'],
      ['Worker', 'Deploying', '5 min ago'],
      ['Postgres', 'Healthy', '10 min ago'],
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('columnheader', { name: 'Resource' })).toBeVisible()
    await expect(canvas.getAllByRole('row')).toHaveLength(4)
    await expect(canvas.getByRole('cell', { name: 'Postgres' })).toBeVisible()
  },
}
