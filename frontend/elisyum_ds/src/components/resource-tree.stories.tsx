import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ResourceTree } from './resource-tree'

const meta = {
  component: ResourceTree,
  title: 'Data/ResourceTree',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ResourceTree>
export default meta
type Story = StoryObj<typeof meta>
export const Workspace: Story = {
  args: {
    items: [
      {
        id: 'apps',
        label: 'Applications',
        kind: 'folder',
        children: [
          { id: 'api', label: 'API service' },
          { id: 'worker', label: 'Worker' },
        ],
      },
      {
        id: 'infra',
        label: 'Infrastructure',
        kind: 'folder',
        children: [{ id: 'db', label: 'Postgres' }],
      },
    ],
    onSelect: fn(),
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Expand Applications' }))
    await userEvent.click(canvas.getByRole('button', { name: 'API service' }))
    await expect(args.onSelect).toHaveBeenCalledWith('api')
  },
  render: (args) => (
    <div className="w-72 rounded-lg border border-border-subtle bg-surface-1 p-3">
      <ResourceTree {...args} />
    </div>
  ),
}
