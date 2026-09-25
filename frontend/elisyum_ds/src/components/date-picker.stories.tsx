import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { DatePicker } from './date-picker'
import { Field } from './field'

const meta = {
  component: DatePicker,
  title: 'Forms/DatePicker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DatePicker>
export default meta
type Story = StoryObj<typeof meta>
export const Release: Story = {
  args: { 'aria-label': 'Release date', onChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Release date' }))
    await userEvent.click(canvas.getByRole('button', { name: 'Today' }))
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const Invalid: Story = {
  render: () => (
    <div className="w-80">
      <Field
        id="release-date"
        label="Release date"
        error="Choose a release date."
      >
        <DatePicker id="release-date" />
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ release: string }>()
    return (
      <DatePicker
        aria-label="Release date"
        {...form.register('release')}
      />
    )
  },
}
