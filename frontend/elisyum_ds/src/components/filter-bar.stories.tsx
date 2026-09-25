import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { FilterBar } from './filter-bar'

const meta = {
  component: FilterBar,
  title: 'Forms/FilterBar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof FilterBar>
export default meta
type Story = StoryObj<typeof meta>
export const ActiveFilters: Story = {
  args: {
    filters: [
      { id: 'status', label: 'Status', value: 'Healthy', onRemove: fn() },
      { id: 'region', label: 'Region', value: 'us-east' },
    ],
  },
  play: async ({ args, canvas }) => {
    await expect(
      canvas.getByRole('button', { name: 'Remove filter Status Healthy' }),
    ).toBeVisible()
    await userEvent.click(
      canvas.getByRole('button', { name: 'Remove filter Status Healthy' }),
    )
    await expect(args.filters?.[0]?.onRemove).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ query: string }>()
    return (
      <div className="w-[34rem]">
        <FilterBar inputProps={form.register('query')} />
      </div>
    )
  },
}
