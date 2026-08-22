import type { Meta, StoryObj } from '@storybook/react'
import { HoverCard } from './hover-card'

const meta = {
  title: 'Overlay/Hover Card',
  component: HoverCard,
  tags: ['autodocs'],
} satisfies Meta<typeof HoverCard>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    trigger: (
      <a href="/services/api" className="text-primary underline">
        aether-api
      </a>
    ),
    children: (
      <div>
        <strong className="block">Aether API</strong>
        <span className="text-body-sm text-muted-foreground">
          Production service, healthy.
        </span>
      </div>
    ),
  },
}
