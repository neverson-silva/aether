import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Forms } from './forms'
import { Input } from './input'

const meta = {
  component: Forms,
  title: 'Foundations/Forms',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Forms>
export default meta
type Story = StoryObj<typeof meta>
export const FieldComposition: Story = {
  args: {
    children: (
      <Input
        aria-label="Service name"
        placeholder="Service name"
      />
    ),
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('heading', { name: 'Form patterns' })).toBeVisible()
    await expect(canvas.getByRole('textbox', { name: 'Service name' })).toBeVisible()
  },
}
