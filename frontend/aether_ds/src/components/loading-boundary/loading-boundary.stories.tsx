import type { Meta, StoryObj } from '@storybook/react'
import { LoadingBoundary } from './loading-boundary'

const meta = {
  title: 'Feedback/Loading Boundary',
  component: LoadingBoundary,
  tags: ['autodocs'],
} satisfies Meta<typeof LoadingBoundary>
export default meta
type Story = StoryObj<typeof meta>
export const Section: Story = {
  args: {
    loading: true,
    variant: 'section',
    children: <div>Loaded content</div>,
  },
}
