import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { CommandRunner } from './command-runner'

const meta = {
  component: CommandRunner,
  title: 'Patterns/CommandRunner',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof CommandRunner>
export default meta
type Story = StoryObj<typeof meta>
export const Terminal: Story = {
  args: { onRun: fn() },
  render: (args) => {
    const form = useForm<{ command: string }>({
      defaultValues: { command: 'aether apps list' },
    })
    return (
      <div className="w-[36rem]">
        <CommandRunner
          {...args}
          inputProps={form.register('command')}
          value={form.watch('command')}
          result="api-service   healthy\nworker        deploying"
        />
      </div>
    )
  },
  play: async ({ args, canvas }) => {
    await expect(canvas.getByRole('region', { name: 'Command result' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Run command' }))
    await expect(args.onRun).toHaveBeenCalled()
  },
}
