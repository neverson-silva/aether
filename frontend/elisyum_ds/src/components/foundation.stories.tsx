import type { Meta, StoryObj } from '@storybook/react-vite'
import { Foundation } from './foundation'
import { Badge } from './badge'

const meta = {
  component: Foundation,
  title: 'Foundations/Foundation',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Foundation>
export default meta
type Story = StoryObj<typeof meta>
export const Tokens: Story = {
  args: {
    children: (
      <div className="flex gap-2">
        <Badge tone="accent">Accent</Badge>
        <Badge tone="success">Success</Badge>
        <Badge tone="warning">Warning</Badge>
        <Badge tone="danger">Danger</Badge>
      </div>
    ),
  },
}
