import type { Meta, StoryObj } from '@storybook/react'
import { EnvironmentSwitcher } from './environment-switcher'

const meta = {
  title: 'Patterns/Environment Switcher',
  component: EnvironmentSwitcher,
  tags: ['autodocs'],
} satisfies Meta<typeof EnvironmentSwitcher>
export default meta
type Story = StoryObj<typeof meta>
export const ProtectedProduction: Story = {
  args: {
    value: 'production',
    warning: 'Production is protected.',
    options: [
      { id: 'development', label: 'Development', kind: 'development' },
      { id: 'staging', label: 'Staging', kind: 'staging' },
      {
        id: 'production',
        label: 'Production',
        kind: 'production',
        protected: true,
      },
    ],
  },
}
