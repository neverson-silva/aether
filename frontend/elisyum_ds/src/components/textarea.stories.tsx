import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { Textarea } from './textarea'

const meta = {
  component: Textarea,
  title: 'Components/Textarea',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Textarea>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { onChange: fn(), placeholder: 'Describe the change', rows: 4 },
  play: async ({ args, canvas }) => {
    const textarea = canvas.getByPlaceholderText('Describe the change')
    await userEvent.type(textarea, 'Deploy the payment service.')
    await expect(textarea).toHaveValue('Deploy the payment service.')
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Invalid: Story = {
  render: () => (
    <div className="w-96">
      <Field
        id="notes"
        label="Release notes"
        error="Release notes are required."
      >
        <Textarea
          id="notes"
          rows={4}
        />
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ notes: string }>({ defaultValues: { notes: '' } })
    return (
      <form
        className="grid w-96 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Field
          id="notes"
          label="Release notes"
        >
          <Textarea
            id="notes"
            {...form.register('notes')}
          />
        </Field>
        <Button type="submit">Save notes</Button>
      </form>
    )
  },
}
