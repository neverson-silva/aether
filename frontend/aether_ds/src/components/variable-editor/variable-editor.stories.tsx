import type { Meta, StoryObj } from '@storybook/react'
import { VariableEditor } from './variable-editor'

const meta = {
  title: 'Patterns/Variable Editor',
  component: VariableEditor,
  tags: ['autodocs'],
} satisfies Meta<typeof VariableEditor>
export default meta
type Story = StoryObj<typeof meta>
export const Secrets: Story = {
  args: {
    variables: [
      {
        id: '1',
        key: 'API_URL',
        value: 'https://api.aether.dev',
        scope: 'production',
      },
      {
        id: '2',
        key: 'API_TOKEN',
        value: 'secret',
        secret: true,
        scope: 'production',
      },
    ],
  },
}
