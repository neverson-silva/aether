import type { Meta, StoryObj } from '@storybook/react-vite'
import { Attachment } from './attachment'

const meta = {
  component: Attachment,
  title: 'Data/Attachment',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Attachment>
export default meta
type Story = StoryObj<typeof meta>
export const Uploads: Story = {
  args: {
    items: [
      {
        id: '1',
        name: 'deployment-manifest.yaml',
        size: '14 KB',
        progress: 74,
        status: 'Uploading',
      },
      { id: '2', name: 'release-notes.md', size: '4 KB', status: 'Ready' },
    ],
  },
}
