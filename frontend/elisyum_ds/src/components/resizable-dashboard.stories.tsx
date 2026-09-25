import type { Meta, StoryObj } from '@storybook/react-vite'
import { ResizableDashboard } from './resizable-dashboard'

const meta = {
  component: ResizableDashboard,
  title: 'Patterns/ResizableDashboard',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ResizableDashboard>
export default meta
type Story = StoryObj<typeof meta>
export const Operations: Story = {
  args: {
    panels: [
      {
        id: 'health',
        title: 'Health',
        content: (
          <p className="text-body text-text-secondary">98.4% successful deployments</p>
        ),
      },
      {
        id: 'activity',
        title: 'Activity',
        content: (
          <p className="text-body text-text-secondary">12 events in the last hour</p>
        ),
      },
      {
        id: 'logs',
        title: 'Logs',
        content: (
          <p className="font-technical text-log text-text-secondary">
            All systems operational
          </p>
        ),
      },
    ],
  },
}
