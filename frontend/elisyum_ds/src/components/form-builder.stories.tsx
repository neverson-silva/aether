import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Button } from './button'
import { FormBuilder } from './form-builder'

type Values = { name: string; runtime: string; automatic: boolean }
const fields = [
  { name: 'name', label: 'Service name', required: true },
  {
    name: 'runtime',
    label: 'Runtime',
    type: 'select' as const,
    options: [
      { value: 'node', label: 'Node.js' },
      { value: 'go', label: 'Go' },
    ],
  },
  { name: 'automatic', label: 'Automatic deploys', type: 'checkbox' as const },
]
const meta = {
  component: FormBuilder,
  title: 'Forms/FormBuilder',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof FormBuilder<Values>>
export default meta
type Story = StoryObj<typeof meta>
export const Service: Story = {
  args: { fields, register: (() => ({})) as never },
  render: () => {
    const form = useForm<Values>({
      defaultValues: { name: '', runtime: '', automatic: true },
    })
    return (
      <form
        className="grid w-96 gap-4"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <FormBuilder
          fields={fields}
          register={form.register}
        />
        <Button type="submit">Create service</Button>
      </form>
    )
  },
  play: async ({ canvas }) => {
    await userEvent.type(
      canvas.getByRole('textbox', { name: 'Service name' }),
      'payments-api',
    )
    await expect(canvas.getByRole('textbox', { name: 'Service name' })).toHaveValue(
      'payments-api',
    )
    await userEvent.click(canvas.getByRole('button', { name: 'Runtime' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Go' }))
    await userEvent.click(canvas.getByRole('checkbox', { name: 'Automatic deploys' }))
    await expect(
      canvas.getByRole('checkbox', { name: 'Automatic deploys' }),
    ).not.toBeChecked()
  },
}
