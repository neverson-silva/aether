import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Badge } from './badge'
import { DataTable, type DataTableColumn } from './data-table'

type Service = Record<string, unknown>
const columns: DataTableColumn<Service>[] = [
  { key: 'name', label: 'Service' },
  {
    key: 'status',
    label: 'Status',
    render: (value) => (
      <Badge tone={String(value) === 'Healthy' ? 'success' : 'accent'}>
        {String(value)}
      </Badge>
    ),
  },
  { key: 'region', label: 'Region' },
]
const meta = {
  component: DataTable,
  title: 'Data/DataTable',
  parameters: { layout: 'centered' },
} satisfies Meta<any>
export default meta
type Story = StoryObj<any>
export const Services: Story = {
  args: {
    columns,
    rows: [
      { name: 'API service', status: 'Healthy', region: 'us-east' },
      { name: 'Worker', status: 'Deploying', region: 'eu-west' },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('columnheader', { name: 'Service' })).toBeVisible()
    await expect(canvas.getByRole('cell', { name: 'API service' })).toBeVisible()
    await expect(canvas.getByText('Healthy')).toBeVisible()
  },
}
export const Empty: Story = {
  args: { columns, rows: [], empty: 'No services match the current filters.' },
  play: async ({ canvas }) => {
    await expect(
      canvas.getByText('No services match the current filters.'),
    ).toBeVisible()
  },
}
