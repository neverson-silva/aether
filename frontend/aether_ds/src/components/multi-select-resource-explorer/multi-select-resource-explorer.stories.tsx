import type { Meta, StoryObj } from '@storybook/react'
import { MultiSelectResourceExplorer } from './multi-select-resource-explorer'

const meta = {
  title: 'Data/Multi-select Resource Explorer',
  component: MultiSelectResourceExplorer,
  tags: ['autodocs'],
} satisfies Meta<typeof MultiSelectResourceExplorer>
export default meta
type Story = StoryObj<typeof meta>
export const Resources: Story = {
  args: {
    nodes: [{ id: 'services', label: 'Services', type: 'folder' }],
    items: [
      { id: 'api', label: 'aether-api', status: 'Healthy' },
      { id: 'web', label: 'aether-web', status: 'Deploying' },
    ],
  },
}
