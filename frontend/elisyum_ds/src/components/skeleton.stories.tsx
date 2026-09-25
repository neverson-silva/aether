import type { Meta, StoryObj } from '@storybook/react-vite'
import { Skeleton } from './skeleton'

const meta = {
  component: Skeleton,
  title: 'Foundations/Skeleton',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Skeleton>
export default meta
type Story = StoryObj<typeof meta>
export const ResourceCard: Story = { args: { className: 'h-24 w-80' } }
