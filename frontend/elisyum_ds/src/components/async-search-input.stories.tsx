import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { AsyncSearchInput } from './async-search-input'

const meta = {
  component: AsyncSearchInput,
  title: 'Forms/AsyncSearchInput',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof AsyncSearchInput>
export default meta
type Story = StoryObj<typeof meta>
const results = [
  { value: 'api', label: 'API service' },
  { value: 'worker', label: 'Worker' },
]
export const Results: Story = {
  args: { onOptionSelect: fn(), placeholder: 'Search services', results },
  play: async ({ canvas, args }) => {
    const input = canvas.getByPlaceholderText('Search services')
    await userEvent.click(input)
    await userEvent.keyboard('{ArrowDown}')
    await expect(input).toHaveAttribute(
      'aria-activedescendant',
      'elysium-async-search-option-api',
    )
    await userEvent.keyboard('{Enter}')
    await expect(args.onOptionSelect).toHaveBeenCalledWith(results[0])
  },
}
export const ReactHookForm: Story = {
  args: { placeholder: 'Search services', results },
  render: (args) => {
    const form = useForm<{ service: string }>()
    return (
      <div className="w-96">
        <AsyncSearchInput
          {...args}
          {...form.register('service')}
        />
      </div>
    )
  },
}
