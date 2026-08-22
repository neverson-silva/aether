import type { Meta, StoryObj } from '@storybook/react'
import { NativeSelect } from './native-select'

const meta = {
  title: 'Forms/Native Select',
  component: NativeSelect,
  tags: ['autodocs'],
} satisfies Meta<typeof NativeSelect>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    label: 'Environment',
    options: [
      { label: 'Production', value: 'production' },
      { label: 'Staging', value: 'staging' },
      { label: 'Development', value: 'development' },
    ],
  },
}
