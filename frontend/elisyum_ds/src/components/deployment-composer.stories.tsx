import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { DeploymentComposer } from './deployment-composer'

type Values = { environment: string; notes: string }
const meta = {
  component: DeploymentComposer,
  title: 'Patterns/DeploymentComposer',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DeploymentComposer<Values>>
export default meta
type Story = StoryObj<typeof meta>
export const Review: Story = {
  args: { register: (() => ({})) as never, onSubmit: fn() },
  render: (args) => {
    const form = useForm<Values>({
      defaultValues: { environment: 'production', notes: '' },
    })
    return (
      <div className="w-[32rem]">
        <DeploymentComposer
          {...args}
          register={form.register}
        />
      </div>
    )
  },
  play: async ({ args, canvas }) => {
    await userEvent.type(
      canvas.getByRole('textbox', { name: 'Release notes' }),
      'Enable blue-green rollout',
    )
    await userEvent.click(canvas.getByRole('button', { name: 'Review deployment' }))
    await expect(args.onSubmit).toHaveBeenCalled()
  },
}
