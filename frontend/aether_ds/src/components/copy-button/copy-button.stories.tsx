import type { Meta, StoryObj } from '@storybook/react'
import { CopyButton } from './copy-button'

const meta = {
  title: 'Components/Copy Button',
  component: CopyButton,
  tags: ['autodocs'],
} satisfies Meta<typeof CopyButton>
export default meta
type Story = StoryObj<typeof meta>
export const ResourceId: Story = {
  args: { value: 'svc_01HZX4AETHER', label: 'Copy ID' },
}
