import type { Meta, StoryObj } from '@storybook/react-vite'
import { LogViewer } from './log-viewer'

const meta = {
  component: LogViewer,
  title: 'Data/LogViewer',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof LogViewer>
export default meta
type Story = StoryObj<typeof meta>
export const Runtime: Story = {
  args: {
    follow: true,
    lines: [
      {
        id: '1',
        timestamp: '10:42:10',
        level: 'INFO',
        message: 'Listening on port 8080',
      },
      { id: '2', timestamp: '10:42:11', level: 'INFO', message: 'Health check passed' },
      {
        id: '3',
        timestamp: '10:42:12',
        level: 'WARN',
        message: 'Upstream latency is elevated',
      },
    ],
  },
  render: (args) => (
    <div className="w-[36rem]">
      <LogViewer {...args} />
    </div>
  ),
}
