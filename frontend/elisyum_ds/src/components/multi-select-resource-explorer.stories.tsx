import { Controller, useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { MultiSelectResourceExplorer } from './multi-select-resource-explorer'

const meta = {
  component: MultiSelectResourceExplorer,
  title: 'Forms/MultiSelectResourceExplorer',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof MultiSelectResourceExplorer>
export default meta
type Story = StoryObj<typeof meta>
const items = [
  { id: 'api', label: 'API service', description: 'production' },
  { id: 'worker', label: 'Worker', description: 'production' },
  { id: 'docs', label: 'Documentation', description: 'staging' },
]
export const Resources: Story = {
  args: { items, onValueChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: /API service/ }))
    await expect(canvas.getByRole('button', { name: /API service/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await expect(args.onValueChange).toHaveBeenCalledWith(['api'])
    await userEvent.type(
      canvas.getByRole('textbox', { name: 'Search resources' }),
      'worker',
    )
    await expect(canvas.getByRole('button', { name: /Worker/ })).toBeVisible()
  },
}
export const ReactHookForm: Story = {
  args: { items },
  render: (args) => {
    const form = useForm<{ resources: string[] }>({ defaultValues: { resources: [] } })
    return (
      <Controller
        control={form.control}
        name="resources"
        render={({ field }) => (
          <div className="w-96">
            <MultiSelectResourceExplorer
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
