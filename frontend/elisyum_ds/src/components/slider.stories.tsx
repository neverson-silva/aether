import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { Field } from './field'
import { Slider } from './slider'

const meta = {
  component: Slider,
  title: 'Components/Slider',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Slider>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    min: 0,
    max: 100,
    defaultValue: 60,
    'aria-label': 'CPU limit',
    onChange: fn(),
  },
  play: async ({ args, canvas }) => {
    const slider = canvas.getByRole('slider', { name: 'CPU limit' })
    await userEvent.click(slider)
    await userEvent.keyboard('{ArrowRight}')
    await expect(slider).toHaveValue('61')
    await expect(args.onChange).toHaveBeenCalled()
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ cpu: number }>({ defaultValues: { cpu: 50 } })
    return (
      <form
        className="grid w-96 gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <Field
          id="cpu"
          label="CPU limit"
        >
          <Slider
            id="cpu"
            {...form.register('cpu', { valueAsNumber: true })}
            min={0}
            max={100}
          />
        </Field>
        <Button type="submit">Save limit</Button>
      </form>
    )
  },
}
