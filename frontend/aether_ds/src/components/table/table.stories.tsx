import type { Meta, StoryObj } from '@storybook/react'
import { Table } from './table'

const meta = {
  title: 'Data/Table',
  component: Table,
  tags: ['autodocs'],
} satisfies Meta<typeof Table>
export default meta
type Story = StoryObj<typeof meta>
export const States: Story = {
  args: { columns: [], data: [] },
  render: () => (
    <Table
      columns={[
        { key: 'service', header: 'Service' },
        { key: 'status', header: 'Status' },
        { key: 'region', header: 'Region' },
      ]}
      data={[
        { service: 'aether-api', status: 'Healthy', region: 'sa-east-1' },
        { service: 'aether-web', status: 'Deploying', region: 'us-east-1' },
      ]}
    />
  ),
}
