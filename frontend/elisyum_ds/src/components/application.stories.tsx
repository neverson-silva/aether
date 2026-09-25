import type { Meta, StoryObj } from '@storybook/react-vite'
import { Application } from './application'
import { Card } from './card'

const meta = {
  component: Application,
  title: 'Patterns/Application',
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof Application>
export default meta
type Story = StoryObj<typeof meta>
export const ProductSurface: Story = {
  args: {
    title: 'Applications',
    description: 'Manage services and deployments.',
    children: <Card>Application content</Card>,
  },
}
