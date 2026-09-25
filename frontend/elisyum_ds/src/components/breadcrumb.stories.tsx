import type { Meta, StoryObj } from '@storybook/react-vite'
import { Breadcrumb } from './breadcrumb'

const meta = {
  component: Breadcrumb,
  title: 'Navigation/Breadcrumb',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Breadcrumb>
export default meta
type Story = StoryObj<typeof meta>
export const ResourcePath: Story = {
  args: {
    items: [
      { label: 'Applications', href: '/applications' },
      { label: 'API service', href: '/applications/api' },
      { label: 'Deployments' },
    ],
  },
}
