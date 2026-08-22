import type { Meta, StoryObj } from '@storybook/react'
import { Typography } from './typography'

const meta = {
  title: 'Foundations/Typography',
  component: Typography,
  tags: ['autodocs'],
  args: { children: 'Aether infrastructure at a glance' },
} satisfies Meta<typeof Typography>
export default meta
type Story = StoryObj<typeof meta>
export const Display: Story = { args: { as: 'h1', level: 'display' } }
export const Heading: Story = { args: { as: 'h2', level: 'heading' } }
export const Body: Story = { args: { level: 'body' } }
export const Label: Story = {
  args: { level: 'label', children: 'PRODUCTION ENVIRONMENT' },
}
export const Code: Story = {
  args: {
    as: 'code',
    level: 'code',
    children: 'aether deploy --environment production',
  },
}
export const States: Story = {
  render: () => (
    <div className="space-y-2">
      <Typography tone="muted">Muted metadata</Typography>
      <Typography tone="primary">Primary link content</Typography>
      <Typography tone="danger">Deployment failed</Typography>
      <Typography truncate>
        Long content that truncates when the available width is too small for
        the complete value.
      </Typography>
    </div>
  ),
}
