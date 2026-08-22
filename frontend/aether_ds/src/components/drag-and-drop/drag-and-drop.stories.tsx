import type { Meta, StoryObj } from '@storybook/react'
import { DragAndDrop } from './drag-and-drop'

const meta = {
  title: 'Patterns/Drag and Drop',
  component: DragAndDrop,
  tags: ['autodocs'],
} satisfies Meta<typeof DragAndDrop>
export default meta
type Story = StoryObj<typeof meta>
export const Upload: Story = {
  args: { label: 'Drop a configuration file here', onDrop: () => undefined },
}
