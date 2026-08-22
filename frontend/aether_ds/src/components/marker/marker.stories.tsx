import type { Meta, StoryObj } from '@storybook/react-vite'
import { Marker } from './marker'

const meta = {
  title: 'Components/Marker',
  component: Marker,
  tags: ['autodocs'],
} satisfies Meta<typeof Marker>
export default meta
type Story = StoryObj<typeof meta>
export const States: Story = {
  args: { children: 'healthy' },
  render: () => (
    <div className="flex gap-2">
      <Marker tone="success">healthy</Marker>
      <Marker tone="warning">degraded</Marker>
      <Marker tone="danger">failed</Marker>
    </div>
  ),
}
