import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { RadioGroup } from './radio-group'

const meta = {
  component: RadioGroup,
  title: 'Components/RadioGroup',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof RadioGroup>
export default meta
type Story = StoryObj<typeof meta>
const options = [
  {
    value: 'standard',
    label: 'Standard',
    description: 'Balanced resources for most services.',
  },
  {
    value: 'performance',
    label: 'Performance',
    description: 'More resources for latency-sensitive work.',
  },
]
export const Default: Story = {
  args: { 'aria-label': 'Plan', onValueChange: fn(), options },
  play: async ({ args, canvas }) => {
    const radio = canvas.getByRole('radio', { name: 'Performance' })
    await userEvent.click(radio)
    await expect(radio).toBeChecked()
    await expect(args.onValueChange).toHaveBeenCalledWith('performance')
  },
}
export const ReactHookForm: Story = {
  args: { options },
  render: () => {
    const form = useForm<{ plan: string }>({ defaultValues: { plan: 'standard' } })
    return (
      <form
        className="grid w-96 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <RadioGroup
          aria-label="Plan"
          inputProps={form.register('plan')}
          options={options}
        />
        <Button type="submit">Continue</Button>
      </form>
    )
  },
}
