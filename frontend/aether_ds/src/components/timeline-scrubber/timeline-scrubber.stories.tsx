import type { Meta, StoryObj } from '@storybook/react'
import { TimelineScrubber } from './timeline-scrubber'

const meta = {
  title: 'Data/Timeline Scrubber',
  component: TimelineScrubber,
  tags: ['autodocs'],
} satisfies Meta<typeof TimelineScrubber>
export default meta
type Story = StoryObj<typeof meta>
export const IncidentReview: Story = {
  args: {
    start: 0,
    end: 100,
    value: 42,
    playback: true,
    timezone: 'UTC',
    markers: [
      { id: '1', position: 30, label: 'Incident', tone: 'danger' },
      { id: '2', position: 60, label: 'Deploy', tone: 'info' },
    ],
  },
}
