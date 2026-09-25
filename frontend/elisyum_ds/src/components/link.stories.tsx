import type { Meta, StoryObj } from '@storybook/react-vite'
import { Link } from './link'

const meta = {
  component: Link,
  title: 'Components/Link',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Link>
export default meta
type Story = StoryObj<typeof meta>
export const Documentation: Story = {
  args: { children: 'Read deployment documentation', href: '/docs' },
}
