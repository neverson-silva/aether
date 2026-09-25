import type { Meta, StoryObj } from '@storybook/react-vite'
import { Typography } from './typography'

const meta = {
  component: Typography,
  title: 'Foundations/Typography',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Typography>
export default meta
type Story = StoryObj<typeof meta>
export const Roles: Story = {
  args: { children: 'Resource deployment overview', role: 'page-title' },
  render: () => (
    <div className="grid gap-3">
      <Typography role="display">Infrastructure, clearly.</Typography>
      <Typography role="page-title">Deployment overview</Typography>
      <Typography role="section-title">Runtime status</Typography>
      <Typography role="body">
        The service is available in the selected environment.
      </Typography>
      <Typography role="supporting">Last updated a moment ago.</Typography>
      <Typography role="code">aether deploy --environment production</Typography>
    </div>
  ),
}
