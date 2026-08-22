import { CloudArrowUp, Rocket } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react'
import { Button } from './button'

const meta = {
  title: 'Components/Button',
  component: Button,
  tags: ['autodocs'],
  args: { children: 'Deploy service' },
  argTypes: {
    variant: {
      control: 'select',
      options: [
        'primary',
        'secondary',
        'ghost',
        'quiet',
        'outline',
        'success',
        'danger',
      ],
    },
    size: { control: 'select', options: ['sm', 'md', 'lg'] },
    loading: { control: 'boolean' },
    disabled: { control: 'boolean' },
  },
} satisfies Meta<typeof Button>

export default meta
type Story = StoryObj<typeof meta>

export const Primary: Story = {}
export const Secondary: Story = { args: { variant: 'secondary' } }
export const Ghost: Story = { args: { variant: 'ghost' } }
export const Quiet: Story = { args: { variant: 'quiet' } }
export const Outline: Story = { args: { variant: 'outline' } }
export const Success: Story = {
  args: { variant: 'success', children: 'Deployment ready' },
}
export const Danger: Story = {
  args: { variant: 'danger', children: 'Delete service' },
}
export const Small: Story = { args: { size: 'sm' } }
export const Large: Story = { args: { size: 'lg' } }
export const Loading: Story = { args: { loading: true } }
export const LoadingWithLabel: Story = {
  args: { loading: true, loadingLabel: 'Deploying service' },
}
export const Disabled: Story = { args: { disabled: true } }
export const WithIcons: Story = {
  args: { icon: Rocket, iconPosition: 'start' },
}
export const WithTrailingIcon: Story = {
  args: { icon: CloudArrowUp, iconPosition: 'end', children: 'Deploy service' },
}
export const FullWidth: Story = { args: { fullWidth: true } }
export const LongLabel: Story = {
  args: { children: 'Create production deployment with approval' },
}
