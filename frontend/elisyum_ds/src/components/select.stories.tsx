import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { Select } from './select'

const meta = {
  component: Select,
  title: 'Components/Select',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Select>
export default meta
type Story = StoryObj<typeof meta>
const options = (
  <>
    <option value="">Choose an environment</option>
    <option value="production">Production</option>
    <option value="staging">Staging</option>
  </>
)
export const Default: Story = {
  args: { 'aria-label': 'Environment', onChange: fn() },
  render: (args) => (
    <div className="w-80">
      <Field
        id="environment"
        label="Environment"
      >
        <Select
          id="environment"
          {...args}
        >
          {options}
        </Select>
      </Field>
    </div>
  ),
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Environment' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Production' }))
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Invalid: Story = {
  render: () => (
    <div className="w-80">
      <Field
        id="environment"
        label="Environment"
        error="Choose an environment."
      >
        <Select id="environment">{options}</Select>
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ environment: string }>({
      defaultValues: { environment: '' },
    })
    return (
      <form
        className="grid w-80 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Field
          id="environment"
          label="Environment"
          error={form.formState.errors.environment?.message}
        >
          <Select
            id="environment"
            {...form.register('environment', { required: 'Choose an environment.' })}
          >
            {options}
          </Select>
        </Field>
        <Button type="submit">Continue</Button>
      </form>
    )
  },
}
