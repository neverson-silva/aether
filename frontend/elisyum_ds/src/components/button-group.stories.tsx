import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from './button'
import { ButtonGroup } from './button-group'

const meta = {
  component: ButtonGroup,
  title: 'Components/ButtonGroup',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ButtonGroup>
export default meta
type Story = StoryObj<typeof meta>
export const Actions: Story = {
  args: {
    children: (
      <>
        <Button size="sm">Deploy</Button>
        <Button
          size="sm"
          tone="ghost"
        >
          Preview
        </Button>
      </>
    ),
  },
}
