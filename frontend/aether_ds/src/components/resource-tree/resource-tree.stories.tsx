import type { Meta, StoryObj } from '@storybook/react'
import { ResourceTree } from './resource-tree'

const meta = {
  title: 'Data/Resource Tree',
  component: ResourceTree,
  tags: ['autodocs'],
} satisfies Meta<typeof ResourceTree>
export default meta
type Story = StoryObj<typeof meta>
export const Resources: Story = {
  args: {
    nodes: [
      {
        id: 'project',
        label: 'Aether Platform',
        children: [
          {
            id: 'services',
            label: 'Services',
            children: [
              { id: 'api', label: 'aether-api', type: 'resource' },
              { id: 'web', label: 'aether-web', type: 'resource' },
            ],
          },
        ],
      },
    ],
  },
}
