import type { Meta, StoryObj } from '@storybook/react-vite'
import { LoadingBoundary } from './loading-boundary'

const meta = {
  component: LoadingBoundary,
  title: 'Feedback/LoadingBoundary',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof LoadingBoundary>
export default meta
type Story = StoryObj<typeof meta>
export const Loading: Story = {
  args: { loading: true, children: <div>Loaded content</div> },
}
export const Loaded: Story = {
  args: {
    loading: false,
    children: (
      <div className="rounded-lg border border-border-subtle p-6 text-body">
        Loaded content
      </div>
    ),
  },
}
