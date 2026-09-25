import type { Meta, StoryObj } from '@storybook/react-vite'
import { Gauge } from './gauge'

const meta = {
  component: Gauge,
  title: 'Data/Gauge',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Gauge>
export default meta
type Story = StoryObj<typeof meta>
export const Capacity: Story = { args: { label: 'CPU capacity', value: 68 } }
