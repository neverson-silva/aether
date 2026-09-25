import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { TimelineScrubber } from './timeline-scrubber'

const meta = {
  component: TimelineScrubber,
  title: 'Data/TimelineScrubber',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof TimelineScrubber>
export default meta
type Story = StoryObj<typeof meta>
export const Playback: Story = {
  args: { min: 0, max: 100, defaultValue: 34 },
  render: (args) => {
    const form = useForm<{ position: number }>({ defaultValues: { position: 34 } })
    return (
      <div className="w-96">
        <TimelineScrubber
          {...args}
          {...form.register('position', { valueAsNumber: true })}
        />
      </div>
    )
  },
}
