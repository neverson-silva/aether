import type { Meta, StoryObj } from '@storybook/react-vite'
import { Changelog } from './changelog'

const meta = {
  component: Changelog,
  title: 'Data/Changelog',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Changelog>
export default meta
type Story = StoryObj<typeof meta>
export const Releases: Story = {
  args: {
    releases: [
      {
        id: '1',
        version: 'v2.4.0',
        title: 'Realtime deployment health',
        description: 'New lifecycle and connection states.',
        timestamp: 'Today',
        status: 'New',
        tone: 'accent',
      },
      {
        id: '2',
        version: 'v2.3.1',
        title: 'Faster log loading',
        timestamp: 'Last week',
        status: 'Fixed',
        tone: 'success',
      },
    ],
  },
  render: (args) => (
    <div className="w-[32rem]">
      <Changelog {...args} />
    </div>
  ),
}
