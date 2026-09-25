import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { FileUpload } from './file-upload'

const meta = {
  component: FileUpload,
  title: 'Forms/FileUpload',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof FileUpload>
export default meta
type Story = StoryObj<typeof meta>
export const Artifacts: Story = {
  args: {
    attachments: [
      { id: '1', name: 'release.yaml', size: '8 KB', progress: 100, status: 'Ready' },
    ],
    onRemove: fn(),
  },
  play: async ({ args, canvas }) => {
    await expect(canvas.getByRole('listitem')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Remove release.yaml' }))
    await expect(args.onRemove).toHaveBeenCalledWith('1')
  },
}
export const ReactHookForm: Story = {
  render: () => {
    const form = useForm<{ artifact: FileList }>()
    return (
      <div className="w-96">
        <FileUpload inputProps={form.register('artifact')} />
      </div>
    )
  },
}
