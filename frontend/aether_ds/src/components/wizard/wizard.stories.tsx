import type { Meta, StoryObj } from '@storybook/react'
import { Wizard } from './wizard'

const meta = {
  title: 'Patterns/Wizard',
  component: Wizard,
  tags: ['autodocs'],
} satisfies Meta<typeof Wizard>
export default meta
type Story = StoryObj<typeof meta>
export const Deployment: Story = {
  args: {
    steps: [
      {
        id: 'source',
        title: 'Source',
        content: (
          <input
            placeholder="Branch or commit"
            className="h-10 w-full rounded border border-border px-3"
          />
        ),
      },
      {
        id: 'review',
        title: 'Review',
        content: <p>Review deployment settings.</p>,
      },
    ],
  },
}
