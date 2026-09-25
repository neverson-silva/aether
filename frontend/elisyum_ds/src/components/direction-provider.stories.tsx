import type { Meta, StoryObj } from '@storybook/react-vite'
import { DirectionProvider } from './direction-provider'
import { Typography } from './typography'

const meta = {
  component: DirectionProvider,
  title: 'Foundations/DirectionProvider',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DirectionProvider>
export default meta
type Story = StoryObj<typeof meta>
export const RightToLeft: Story = {
  args: {
    direction: 'rtl',
    children: <Typography>Direction-aware content</Typography>,
  },
}
