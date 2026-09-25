import type { Meta, StoryObj } from '@storybook/react-vite'
import { ProgressRing } from './progress-ring'

const meta = {
  component: ProgressRing,
  title: 'Components/ProgressRing',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ProgressRing>
export default meta
type Story = StoryObj<typeof meta>
export const Deploying: Story = { args: { label: 'Deployment progress', value: 72 } }
