import type { Meta, StoryObj } from '@storybook/react-vite'
import { DiffViewer } from './diff-viewer'

const meta = {
  component: DiffViewer,
  title: 'Data/DiffViewer',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DiffViewer>
export default meta
type Story = StoryObj<typeof meta>
export const ConfigurationChange: Story = {
  args: {
    lines: [
      { id: '1', type: 'unchanged', content: 'replicas: 2' },
      { id: '2', type: 'removed', content: 'memory: 512Mi' },
      { id: '3', type: 'added', content: 'memory: 1Gi' },
    ],
  },
  render: (args) => (
    <div className="w-[36rem]">
      <DiffViewer {...args} />
    </div>
  ),
}
