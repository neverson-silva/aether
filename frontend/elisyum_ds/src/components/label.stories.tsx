import type { Meta, StoryObj } from '@storybook/react-vite'
import { Label } from './label'

const meta = {
  component: Label,
  title: 'Components/Label',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Label>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { htmlFor: 'service-name', children: 'Service name' },
}
