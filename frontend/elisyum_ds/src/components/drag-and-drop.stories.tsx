import type { Meta, StoryObj } from '@storybook/react-vite'
import { DragAndDrop } from './drag-and-drop'

const meta = {
  component: DragAndDrop,
  title: 'Forms/DragAndDrop',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DragAndDrop>
export default meta
type Story = StoryObj<typeof meta>
export const Files: Story = { args: { accept: '.yaml,.json' } }
