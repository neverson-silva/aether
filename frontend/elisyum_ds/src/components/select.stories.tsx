import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { Modal } from './modal'
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
    await userEvent.click(canvas.getByRole('combobox', { name: 'Environment' }))
    const productionOption = [...document.querySelectorAll('[role="option"]')].find(
      (option) => option.textContent === 'Production',
    )
    if (!(productionOption instanceof HTMLElement)) throw new Error('Production option was not rendered.')
    await userEvent.click(productionOption)
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const InDialog: Story = {
  render: () => (
    <Modal
      trigger="Open dialog"
      title="Storage destination"
      footer={<Button>Save destination</Button>}
    >
      <Field id="destination-type" label="Destination type" required>
        <Select value="production">
          <option value="production">Production</option>
          <option value="staging">Staging</option>
          <option value="google-drive">Google Drive</option>
        </Select>
      </Field>
    </Modal>
  ),
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open dialog' }))
    await userEvent.click(canvas.getByRole('combobox', { name: 'Destination type' }))
    await expect(
      [...document.querySelectorAll('[role="option"]')].some(
        (option) => option.textContent === 'Google Drive',
      ),
    ).toBe(true)
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
