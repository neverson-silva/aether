import type { Meta, StoryObj } from '@storybook/react-vite'
import { Attachment } from './attachment'

const meta = {
  title: 'Components/Attachment',
  component: Attachment,
  tags: ['autodocs'],
} satisfies Meta<typeof Attachment>
export default meta
type Story = StoryObj<typeof meta>
export const Empty: Story = {
  args: { label: 'Build artifacts', accept: '.zip,.tar.gz' },
}
export const Uploading: Story = {
  args: {
    items: [
      {
        id: '1',
        name: 'aether-build.zip',
        size: 2400000,
        status: 'uploading',
        progress: 62,
      },
    ],
  },
}
export const Failed: Story = {
  args: {
    items: [
      {
        id: '1',
        name: 'secrets.env',
        status: 'error',
        error: 'Upload failed.',
      },
    ],
  },
}
