import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Button } from './button'
import { VariableEditor } from './variable-editor'

type Values = { API_URL: string; API_TOKEN: string }
const variables = [
  { name: 'API_URL', description: 'Public service endpoint' },
  { name: 'API_TOKEN', description: 'Encrypted secret', secret: true },
]
const meta = {
  component: VariableEditor,
  title: 'Forms/VariableEditor',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof VariableEditor<Values>>
export default meta
type Story = StoryObj<typeof meta>
export const Environment: Story = {
  args: { register: (() => ({})) as never, variables },
  render: () => {
    const form = useForm<Values>({
      defaultValues: { API_URL: 'https://api.example.com', API_TOKEN: '' },
    })
    return (
      <form
        className="grid w-[32rem] gap-4"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <VariableEditor
          register={form.register}
          variables={variables}
        />
        <Button type="submit">Save variables</Button>
      </form>
    )
  },
  play: async ({ canvas }) => {
    const token = canvas.getByLabelText('API_TOKEN')
    await expect(token).toHaveAttribute('type', 'password')
    await userEvent.click(canvas.getByRole('button', { name: 'Show API_TOKEN' }))
    await expect(token).toHaveAttribute('type', 'text')
  },
}
