import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { Input } from './input'

const meta = {
  component: Input,
  title: 'Components/Input',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Input>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { onChange: fn(), placeholder: 'service-api' },
  play: async ({ args, canvas }) => {
    const input = canvas.getByPlaceholderText('service-api')
    await userEvent.type(input, 'payments-api')
    await expect(input).toHaveValue('payments-api')
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Invalid: Story = {
  render: () => (
    <div className="w-80">
      <Field
        id="service-name"
        label="Service name"
        error="A service name is required."
      >
        <Input id="service-name" />
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ name: string }>({ defaultValues: { name: '' } })
    return (
      <form
        className="grid w-80 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Field
          id="name"
          label="Service name"
          error={form.formState.errors.name?.message}
        >
          <Input
            id="name"
            {...form.register('name', { required: 'A service name is required.' })}
            aria-invalid={Boolean(form.formState.errors.name)}
          />
        </Field>
        <Button type="submit">Validate</Button>
      </form>
    )
  },
}
