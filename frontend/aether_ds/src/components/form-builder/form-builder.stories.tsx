import type { Meta, StoryObj } from '@storybook/react'
import { FormBuilder } from './form-builder'

const meta = {
  title: 'Patterns/Form Builder',
  component: FormBuilder,
  tags: ['autodocs'],
} satisfies Meta<typeof FormBuilder>
export default meta
type Story = StoryObj<typeof meta>
export const Preview: Story = {
  args: {
    preview: true,
    fields: [
      { id: 'name', label: 'Service name', type: 'text', required: true },
      {
        id: 'token',
        label: 'API token',
        type: 'secret',
        description: 'Stored securely.',
      },
    ],
  },
}
