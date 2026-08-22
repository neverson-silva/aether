import type { Meta, StoryObj } from '@storybook/react'
import { DataTable } from './data-table'

const meta = {
  title: 'Data/Data Table',
  component: DataTable,
  tags: ['autodocs'],
} satisfies Meta<typeof DataTable>
export default meta
type Story = StoryObj<typeof meta>
export const Selectable: Story = {
  args: { columns: [], data: [], rowId: () => '' },
  render: () => (
    <DataTable
      rowId={(row: { id: string }) => row.id}
      data={[
        { id: 'api', service: 'aether-api', replicas: 3 },
        { id: 'web', service: 'aether-web', replicas: 2 },
      ]}
      columns={[
        {
          id: 'service',
          header: 'Service',
          accessor: (row) => row.service,
          sortValue: (row) => row.service,
        },
        {
          id: 'replicas',
          header: 'Replicas',
          accessor: (row) => row.replicas,
          sortValue: (row) => row.replicas,
        },
      ]}
      selectable
    />
  ),
}
