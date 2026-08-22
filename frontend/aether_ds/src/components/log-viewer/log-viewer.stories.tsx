import type { Meta, StoryObj } from '@storybook/react'
import { LogViewer } from './log-viewer'

const meta = {
  title: 'Data/Log Viewer',
  component: LogViewer,
  tags: ['autodocs'],
} satisfies Meta<typeof LogViewer>
export default meta
type Story = StoryObj<typeof meta>
export const Runtime: Story = {
  args: {
    lines: [
      {
        id: '1',
        timestamp: '12:40:01',
        severity: 'info',
        message: 'Starting deployment pipeline',
      },
      {
        id: '2',
        timestamp: '12:40:04',
        severity: 'success',
        message: 'Health check passed',
      },
      {
        id: '3',
        timestamp: '12:40:05',
        severity: 'warning',
        message: 'Waiting for replica readiness',
      },
    ],
  },
}
