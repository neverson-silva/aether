import type { Meta, StoryObj } from '@storybook/react'
import { Carousel } from './carousel'

const meta = {
  title: 'Components/Carousel',
  component: Carousel,
  tags: ['autodocs'],
} satisfies Meta<typeof Carousel>
export default meta
type Story = StoryObj<typeof meta>
export const Slides: Story = {
  args: {
    items: [
      <div className="flex h-48 items-center justify-center bg-primary/10 text-headline-sm">
        Overview
      </div>,
      <div className="flex h-48 items-center justify-center bg-status-success-container/20 text-headline-sm">
        Healthy
      </div>,
      <div className="flex h-48 items-center justify-center bg-status-warning-container/20 text-headline-sm">
        Maintenance
      </div>,
    ],
    autoplay: true,
  },
}
