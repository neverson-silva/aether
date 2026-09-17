import type { Meta, StoryObj } from '@storybook/react'
import { Select } from './select'

const options = Array.from({ length: 500 }, (_, index) => ({
  value: `option-${index + 1}`,
  label: `Option ${index + 1}`,
}))

const meta = {
  title: 'Forms/Select',
  component: Select,
  tags: ['autodocs'],
} satisfies Meta<typeof Select>

export default meta
type Story = StoryObj<typeof meta>

export const LongOptionList: Story = {
  render: () => (
    <div className="min-h-[180vh] space-y-[60vh] p-8">
      <Select label="Top trigger" options={options} />
      <Select label="Center trigger" options={options} />
      <Select label="Bottom trigger" options={options} />
    </div>
  ),
}
