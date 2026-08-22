import type { Meta, StoryObj } from '@storybook/react'
import { DiffViewer } from './diff-viewer'

const meta = {
  title: 'Data/Diff Viewer',
  component: DiffViewer,
  tags: ['autodocs'],
} satisfies Meta<typeof DiffViewer>
export default meta
type Story = StoryObj<typeof meta>
export const Unified: Story = {
  args: {
    lines: [
      {
        id: '1',
        type: 'context',
        oldLine: 1,
        newLine: 1,
        content: 'service: api',
      },
      { id: '2', type: 'removal', oldLine: 2, content: 'replicas: 2' },
      { id: '3', type: 'addition', newLine: 2, content: 'replicas: 3' },
    ],
    collapseContext: true,
  },
}
