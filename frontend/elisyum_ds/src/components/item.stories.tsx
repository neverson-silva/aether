import { Cube, Database, Globe } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { Badge } from './badge'
import { Item, ItemGroup } from './item'

const meta = {
  component: Item,
  title: 'Components/Item',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Item>
export default meta
type Story = StoryObj<typeof meta>
export const ResourceList: Story = {
  args: { title: 'API service', children: null },
  render: () => (
    <ItemGroup className="w-96">
      <Item
        leading={<Cube size={20} />}
        title="API service"
        description="production / us-east"
        trailing={<Badge tone="success">Healthy</Badge>}
      />
      <Item
        leading={<Database size={20} />}
        selected
        title="Postgres"
        description="primary cluster"
        trailing={<Badge>Ready</Badge>}
      />
      <Item
        leading={<Globe size={20} />}
        title="Public domain"
        description="api.example.com"
      />
    </ItemGroup>
  ),
}
