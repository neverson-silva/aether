import type { Meta, StoryObj } from '@storybook/react'
import { ResourcePicker } from './resource-picker'

const meta = {
  title: 'Forms/Resource Picker',
  component: ResourcePicker,
  tags: ['autodocs'],
} satisfies Meta<typeof ResourcePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Services: Story = {
  args: {
    nodes: [
      {
        id: 'services',
        label: 'Services',
        children: [
          { id: 'api', label: 'aether-api', type: 'resource' },
          { id: 'web', label: 'aether-web', type: 'resource' },
        ],
      },
    ],
    recent: [{ id: 'api', label: 'aether-api', type: 'resource' }],
  },
}
