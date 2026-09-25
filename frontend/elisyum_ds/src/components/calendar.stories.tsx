import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Calendar } from './calendar'
import { Field } from './field'

const meta = {
  component: Calendar,
  title: 'Forms/Calendar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Calendar>
export default meta
type Story = StoryObj<typeof meta>
export const Date: Story = {
  args: { 'aria-label': 'Release date', onChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Release date' }))
    await expect(canvas.getByRole('dialog', { name: 'Choose date' })).toBeVisible()
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
        <Calendar id="release-date" />
      </Field>
    </div>
  ),
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ date: string }>()
    return (
      <Calendar
        aria-label="Release date"
        {...form.register('date')}
      />
    )
  },
}
