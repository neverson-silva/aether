import type { Meta, StoryObj } from '@storybook/react'
import { DirectionProvider } from './direction-provider'

const meta = {
  title: 'Foundations/Direction Provider',
  component: DirectionProvider,
  tags: ['autodocs'],
} satisfies Meta<typeof DirectionProvider>
export default meta
type Story = StoryObj<typeof meta>
export const RTL: Story = {
  args: {
    direction: 'rtl',
    children: (
      <div className="rounded border border-border p-4">
        RTL content direction
      </div>
    ),
  },
}
