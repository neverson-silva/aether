import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { NumberField } from './number-field'

const meta = {
  component: NumberField,
  title: 'Components/NumberField',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof NumberField>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { min: 1, max: 12, defaultValue: 3, 'aria-label': 'Replicas', onChange: fn() },
  play: async ({ args, canvas }) => {
    const input = canvas.getByRole('spinbutton', { name: 'Replicas' })
    await userEvent.clear(input)
    await userEvent.type(input, '5')
    await expect(input).toHaveValue(5)
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ replicas: number }>({ defaultValues: { replicas: 2 } })
    return (
      <form
        className="grid w-80 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Field
          id="replicas"
          label="Replicas"
        >
          <NumberField
            id="replicas"
            {...form.register('replicas', {
              valueAsNumber: true,
              min: { value: 1, message: 'At least one replica is required.' },
            })}
            min={1}
            max={12}
          />
        </Field>
        <Button type="submit">Apply</Button>
      </form>
    )
  },
}
