import type { Meta, StoryObj } from '@storybook/react'
import { AspectRatio } from './aspect-ratio'

const meta = {
  title: 'Foundations/Aspect Ratio',
  component: AspectRatio,
  tags: ['autodocs'],
} satisfies Meta<typeof AspectRatio>
export default meta
type Story = StoryObj<typeof meta>
export const Preview: Story = {
  args: {
    ratio: 16 / 9,
    children: (
      <div className="flex h-full items-center justify-center bg-surface-container text-body-sm text-muted-foreground">
        Media preview
      </div>
    ),
  },
}
