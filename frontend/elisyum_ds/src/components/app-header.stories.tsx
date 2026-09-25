import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Button } from './button'
import { AppHeader } from './app-header'

const meta = {
  component: AppHeader,
  title: 'Navigation/AppHeader',
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof AppHeader>
export default meta
type Story = StoryObj<typeof meta>
export const Resource: Story = {
  args: {
    title: 'API service',
    description: 'Production application and deployment history.',
    breadcrumbs: [
      { label: 'Applications', href: '/applications' },
      { label: 'API service' },
    ],
    actions: <Button>Deploy</Button>,
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('heading', { name: 'API service' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Deploy' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Deploy' }))
  },
}
