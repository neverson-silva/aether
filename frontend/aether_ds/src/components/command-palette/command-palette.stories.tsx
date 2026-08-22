import { Rocket, TerminalWindow } from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react'
import { CommandPalette } from './command-palette'

const meta = {
  title: 'Navigation/Command Palette',
  component: CommandPalette,
  tags: ['autodocs'],
} satisfies Meta<typeof CommandPalette>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: {
    trigger: (
      <button
        type="button"
        className="rounded-md border border-border px-3 py-2"
      >
        Search commands
      </button>
    ),
    items: [
      {
        id: 'deploy',
        label: 'Deploy service',
        description: 'Start a production deployment',
        icon: <Rocket size={18} />,
        shortcut: 'D',
      },
      {
        id: 'logs',
        label: 'Open logs',
        description: 'Inspect runtime output',
        icon: <TerminalWindow size={18} />,
        shortcut: 'L',
      },
    ],
  },
}
