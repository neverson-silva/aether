import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { DataGrid } from './data-grid'

const meta = {
  component: DataGrid,
  title: 'Data/DataGrid',
  parameters: { layout: 'centered' },
} satisfies Meta<any>
export default meta
type Story = StoryObj<any>
export const Runtime: Story = {
  args: {
    caption: 'Runtime data',
    columns: [
      { key: 'name', label: 'Service' },
      { key: 'cpu', label: 'CPU' },
    ],
    rows: [
      { name: 'API service', cpu: '42%' },
      { name: 'Worker', cpu: '18%' },
    ],
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('region', { name: 'Runtime data' })).toBeVisible()
    await expect(canvas.getByRole('cell', { name: '42%' })).toBeVisible()
  },
}
