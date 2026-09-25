import type { Meta, StoryObj } from '@storybook/react-vite'
import { Input } from './input'
import { Field } from './field'

const meta = {
  component: Field,
  title: 'Components/Field',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Field>
export default meta
type Story = StoryObj<typeof meta>
export const Valid: Story = {
  args: { children: null },
  render: () => (
    <div className="w-80">
      <Field
        id="service-name"
        label="Service name"
        description="Use a stable name for this service."
      >
        <Input id="service-name" />
      </Field>
    </div>
  ),
}
export const Invalid: Story = {
  args: { children: null },
  render: () => (
    <div className="w-80">
      <Field
        id="service-name"
        label="Service name"
        required
        error="A service name is required."
      >
        <Input id="service-name" />
      </Field>
    </div>
  ),
}
