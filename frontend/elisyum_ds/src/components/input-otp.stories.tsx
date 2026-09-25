import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Button } from './button'
import { InputOTP } from './input-otp'

const meta = {
  component: InputOTP,
  title: 'Components/InputOTP',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof InputOTP>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { label: 'Verification code', length: 6, onChange: fn() },
  play: async ({ args, canvas }) => {
    const firstDigit = canvas.getByRole('textbox', {
      name: 'Verification code digit 1 of 6',
    })
    await userEvent.type(firstDigit, '4')
    await expect(firstDigit).toHaveValue('4')
    await expect(args.onChange).toHaveBeenCalledWith('4')
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ code: string }>({ defaultValues: { code: '' } })
    return (
      <form
        className="grid gap-3"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <InputOTP
          inputProps={form.register('code', { required: true })}
          label="Verification code"
        />
        <Button type="submit">Verify</Button>
      </form>
    )
  },
}
