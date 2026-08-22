import type { Meta, StoryObj } from '@storybook/react'
import { DataGrid } from './data-grid'

const meta = {
  title: 'Data/Data Grid',
  component: DataGrid,
  tags: ['autodocs'],
} satisfies Meta<typeof DataGrid>
export default meta
type Story = StoryObj<typeof meta>
export const DenseResources: Story = {
  args: { columns: [], data: [], rowId: () => '' },
  render: () => (
    <DataGrid
      rowId={(row: { id: string }) => row.id}
      data={[
        { id: '1', name: 'api', status: 'healthy' },
        { id: '2', name: 'worker', status: 'deploying' },
      ]}
      columns={[
        { id: 'name', header: 'Name', accessor: (row) => row.name },
        { id: 'status', header: 'Status', accessor: (row) => row.status },
      ]}
    />
  ),
}
