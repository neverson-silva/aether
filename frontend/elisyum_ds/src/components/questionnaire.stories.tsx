import { Controller, useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Questionnaire } from './questionnaire'

const meta = {
  component: Questionnaire,
  title: 'Forms/Questionnaire',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Questionnaire>
export default meta
type Story = StoryObj<typeof meta>
const questions = [
  {
    id: 'traffic',
    prompt: 'What best describes your traffic?',
    options: [
      { value: 'steady', label: 'Steady traffic' },
      { value: 'bursty', label: 'Bursty traffic' },
    ],
  },
]
export const Discovery: Story = {
  args: { questions, onValueChange: fn() },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('radio', { name: 'Bursty traffic' }))
    await expect(args.onValueChange).toHaveBeenCalledWith('traffic', 'bursty')
  },
}
export const ReactHookForm: Story = {
  args: { questions },
  render: (args) => {
    const form = useForm<{ traffic: string }>({ defaultValues: { traffic: 'steady' } })
    return (
      <Controller
        control={form.control}
        name="traffic"
        render={({ field }) => (
          <div className="w-96">
            <Questionnaire
              {...args}
              onValueChange={(_, value) => field.onChange(value)}
              values={{ traffic: field.value }}
            />
          </div>
        )}
      />
    )
  },
}
