import type { Meta, StoryObj } from '@storybook/react'
import { Changelog } from './changelog'

const meta = {
  title: 'Patterns/Changelog',
  component: Changelog,
  tags: ['autodocs'],
} satisfies Meta<typeof Changelog>
export default meta
type Story = StoryObj<typeof meta>
export const Releases: Story = {
  args: {
    releases: [
      {
        id: '1',
        version: 'v2.4.0',
        title: 'Environment protection controls',
        summary: 'Add approval policies for production.',
        date: 'August 21, 2026',
        category: 'feature',
        impact: 'medium',
        unread: true,
        details:
          'Teams can now require approval before sensitive deployment actions.',
        migration: 'Review existing production policies after upgrading.',
      },
    ],
  },
}
