import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { CodeEditorLite } from './code-editor-lite'

const meta = {
  component: CodeEditorLite,
  title: 'Data/CodeEditorLite',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof CodeEditorLite>
export default meta
type Story = StoryObj<typeof meta>
export const Configuration: Story = {
  args: { defaultValue: '{\n  "replicas": 2\n}', language: 'json' },
  render: (args) => {
    const form = useForm<{ config: string }>()
    return (
      <div className="w-[36rem]">
        <CodeEditorLite
          {...args}
          {...form.register('config')}
        />
      </div>
    )
  },
}
