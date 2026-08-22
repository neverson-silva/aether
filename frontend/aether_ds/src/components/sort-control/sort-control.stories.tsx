import type { Meta, StoryObj } from '@storybook/react'
import { SortControl } from './sort-control'

const meta = {
  title: 'Data/Sort Control',
  component: SortControl,
  tags: ['autodocs'],
} satisfies Meta<typeof SortControl>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = { args: { label: 'Updated' } }
