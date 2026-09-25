import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Combobox } from './combobox'

const meta = {
  component: Combobox,
  title: 'Forms/Combobox',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Combobox>
export default meta
type Story = StoryObj<typeof meta>
const options = [
  { value: 'node', label: 'Node.js' },
  { value: 'go', label: 'Go' },
  { value: 'python', label: 'Python' },
]
export const Runtimes: Story = {
  args: {
    'aria-label': 'Runtime',
    onChange: fn(),
    options,
    placeholder: 'Choose a runtime',
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Show options' }))
    await userEvent.click(canvas.getByRole('option', { name: 'Go' }))
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  args: { options },
  render: (args) => {
    const form = useForm<{ runtime: string }>()
    return (
      <div className="w-80">
        <Combobox
          {...args}
          {...form.register('runtime')}
        />
      </div>
    )
  },
}
