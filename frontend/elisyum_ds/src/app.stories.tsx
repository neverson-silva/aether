import type { Meta, StoryObj } from '@storybook/react-vite'
import { ElisyumPreview } from './app'

const meta = {
  title: 'Foundations/Elisyum Preview',
  component: ElisyumPreview,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof ElisyumPreview>

export default meta

type Story = StoryObj<typeof meta>

export const Default: Story = {}
