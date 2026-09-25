import type { Meta, StoryObj } from '@storybook/react-vite'
import { InlineError } from './inline-error'

const meta = {
  component: InlineError,
  title: 'Feedback/InlineError',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof InlineError>
export default meta
type Story = StoryObj<typeof meta>
export const InvalidField: Story = { args: { children: 'A valid domain is required.' } }
