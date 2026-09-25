import { Controller, useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { ResourcePicker } from './resource-picker'

const meta = {
  component: ResourcePicker,
  title: 'Forms/ResourcePicker',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ResourcePicker>
export default meta
type Story = StoryObj<typeof meta>
const options = [
  { value: 'api', label: 'API service' },
  { value: 'worker', label: 'Worker' },
]
export const Resources: Story = {
  args: { label: 'Resource', options, onValueChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Show options' }))
    await userEvent.click(canvas.getByRole('option', { name: 'API service' }))
    await expect(args.onValueChange).toHaveBeenCalledWith('api')
  },
}
export const ReactHookForm: Story = {
  args: { label: 'Resource', options },
  render: (args) => {
    const form = useForm<{ resource: string }>({ defaultValues: { resource: '' } })
    return (
      <Controller
        control={form.control}
        name="resource"
        render={({ field }) => (
          <div className="w-80">
            <ResourcePicker
              {...args}
              onValueChange={field.onChange}
              value={field.value}
            />
          </div>
        )}
      />
    )
  },
}
