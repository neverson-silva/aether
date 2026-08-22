import type { Meta, StoryObj } from '@storybook/react'
import { ResizableDashboard } from './resizable-dashboard'

const meta = {
  title: 'Patterns/Resizable Dashboard',
  component: ResizableDashboard,
  tags: ['autodocs'],
} satisfies Meta<typeof ResizableDashboard>
export default meta
type Story = StoryObj<typeof meta>
export const Widgets: Story = {
  args: {
    widgets: [
      {
        id: 'deploys',
        title: 'Deployments',
        content: <div className="text-headline-sm">42</div>,
        colSpan: 2,
      },
      {
        id: 'health',
        title: 'Health',
        content: (
          <div className="text-status-success">All systems operational</div>
        ),
      },
      {
        id: 'latency',
        title: 'Latency',
        content: <div className="text-headline-sm">124ms</div>,
      },
    ],
  },
}
