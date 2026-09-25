import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { SelectSearch } from './select-search'

const meta = {
  component: SelectSearch,
  title: 'Forms/SelectSearch',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof SelectSearch>
export default meta
type Story = StoryObj<typeof meta>
const options = [
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
]
export const Environments: Story = {
  args: { options, onChange: fn(), placeholder: 'Search environments' },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Show options' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Production' }))
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  args: { options },
  render: (args) => {
    const form = useForm<{ environment: string }>()
    return (
      <div className="w-80">
        <SelectSearch
          {...args}
          {...form.register('environment')}
        />
      </div>
    )
  },
}
