import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from '../button/button'
import { useToast } from '../toast/toast'
import { Sonner } from './sonner'

function Example() {
  const toast = useToast()
  return (
    <Button
      onClick={() =>
        toast.add({
          title: 'Deployment queued',
          description: 'The runner will start shortly.',
          tone: 'success',
        })
      }
    >
      Show toast
    </Button>
  )
}
const meta = {
  title: 'Feedback/Sonner',
  component: Sonner,
  tags: ['autodocs'],
} satisfies Meta<typeof Sonner>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {
  args: { children: <span /> },
  render: () => (
    <Sonner>
      <Example />
    </Sonner>
  ),
}
