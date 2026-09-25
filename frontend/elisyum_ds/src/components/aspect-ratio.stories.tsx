import type { Meta, StoryObj } from '@storybook/react-vite'
import { AspectRatio } from './aspect-ratio'

const meta = {
  component: AspectRatio,
  title: 'Foundations/AspectRatio',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof AspectRatio>
export default meta
type Story = StoryObj<typeof meta>
export const Preview: Story = {
  args: {
    children: (
      <div className="grid h-full place-items-center bg-action-soft text-body text-action-strong">
        16:9 preview
      </div>
    ),
  },
}
