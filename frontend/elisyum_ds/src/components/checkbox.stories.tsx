import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Checkbox } from './checkbox'

const meta = {
  component: Checkbox,
  title: 'Components/Checkbox',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Checkbox>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { 'aria-label': 'Enable automatic deploys', onChange: fn() },
  play: async ({ args, canvas }) => {
    const checkbox = canvas.getByRole('checkbox', { name: 'Enable automatic deploys' })
    await userEvent.click(checkbox)
    await expect(checkbox).toBeChecked()
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ automatic: boolean }>({ defaultValues: { automatic: true } })
    return (
      <form
        className="flex items-center gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Checkbox {...form.register('automatic')} />
        <span className="text-body">Enable automatic deploys</span>
        <Button
          type="submit"
          size="sm"
        >
          Save
        </Button>
      </form>
    )
  },
}
