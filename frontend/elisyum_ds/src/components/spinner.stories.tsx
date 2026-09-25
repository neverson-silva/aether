import type { Meta, StoryObj } from '@storybook/react-vite'
import { Spinner } from './spinner'

const meta = {
  component: Spinner,
  title: 'Foundations/Spinner',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Spinner>
export default meta
type Story = StoryObj<typeof meta>
export const Sizes: Story = {
  args: { label: 'Loading resource' },
  render: () => (
    <div className="flex items-center gap-5">
      <Spinner size="sm" />
      <Spinner />
      <Spinner size="lg" />
    </div>
  ),
}
