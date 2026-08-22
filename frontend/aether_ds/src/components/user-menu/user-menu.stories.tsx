import type { Meta, StoryObj } from '@storybook/react'
import { UserMenu } from './user-menu'

const meta = {
  title: 'Navigation/User Menu',
  component: UserMenu,
  tags: ['autodocs'],
} satisfies Meta<typeof UserMenu>
export default meta
type Story = StoryObj<typeof meta>
export const WorkspaceSwitcher: Story = {
  args: {
    user: { name: 'Neverson Silva', email: 'neverson@aether.dev' },
    currentWorkspace: 'aether',
    workspaces: [
      {
        id: 'aether',
        label: 'Aether Platform',
        description: 'Production workspace',
      },
      { id: 'labs', label: 'Aether Labs', description: 'Experiments' },
    ],
  },
}
