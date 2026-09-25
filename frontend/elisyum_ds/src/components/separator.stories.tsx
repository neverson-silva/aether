import type { Meta, StoryObj } from '@storybook/react-vite'
import { Separator } from './separator'

const meta = {
  component: Separator,
  title: 'Foundations/Separator',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Separator>
export default meta
type Story = StoryObj<typeof meta>
export const Orientations: Story = {
  render: () => (
    <div className="flex h-12 w-64 items-center gap-4">
      <span>Primary</span>
      <Separator orientation="vertical" />
      <span>Secondary</span>
    </div>
  ),
}
