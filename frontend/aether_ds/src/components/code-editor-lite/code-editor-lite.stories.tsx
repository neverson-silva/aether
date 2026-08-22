import type { Meta, StoryObj } from '@storybook/react-vite'
import { CodeEditorLite } from './code-editor-lite'

const meta = {
  title: 'Components/Code Editor Lite',
  component: CodeEditorLite,
  tags: ['autodocs'],
} satisfies Meta<typeof CodeEditorLite>
export default meta
type Story = StoryObj<typeof meta>
export const Editable: Story = {
  args: {
    filename: 'aether.config.ts',
    language: 'typescript',
    defaultValue: "export default {\n  region: 'sa-east-1',\n  replicas: 3,\n}",
  },
}
export const ReadOnly: Story = {
  args: {
    readOnly: true,
    defaultValue: 'kubectl get deployments --watch',
    language: 'shell',
  },
}
