import type { Meta, StoryObj } from '@storybook/react'
import { FilterBar } from './filter-bar'

const meta = {
  title: 'Forms/Filter Bar',
  component: FilterBar,
  tags: ['autodocs'],
} satisfies Meta<typeof FilterBar>
export default meta
type Story = StoryObj<typeof meta>
export const ActiveFilters: Story = {
  args: {
    activeCount: 2,
    filters: [
      { id: 'status', label: 'Status', value: 'Healthy' },
      { id: 'environment', label: 'Environment', value: 'Production' },
    ],
  },
}
