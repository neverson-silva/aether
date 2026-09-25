import type { Meta, StoryObj } from '@storybook/react-vite'
import { Card } from './card'

const meta = {
  component: Card,
  title: 'Components/Card',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Card>
export default meta
type Story = StoryObj<typeof meta>
export const Surface: Story = {
  args: {
    children: (
      <div className="grid gap-1">
        <h3 className="text-section-title text-text-primary">Production</h3>
        <p className="text-supporting text-text-tertiary">
          12 services in this environment.
        </p>
      </div>
    ),
  },
}
export const Interactive: Story = {
  args: {
    children: (
      <span className="text-body text-text-primary">Open resource details</span>
    ),
    interactive: true,
  },
}
