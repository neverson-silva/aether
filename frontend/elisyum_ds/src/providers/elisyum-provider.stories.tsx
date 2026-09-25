import type { Meta, StoryObj } from '@storybook/react-vite'
import { ElisyumPreview } from '../app'
import { ElisyumProvider } from './elisyum-provider'

const meta = {
  title: 'Foundations/ElisyumProvider',
  component: ElisyumProvider,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof ElisyumProvider>

export default meta

type Story = StoryObj<typeof meta>

export const Dark: Story = {
  args: {
    theme: 'dark',
    density: 'standard',
    children: <ElisyumPreview />,
  },
}

export const Light: Story = {
  args: {
    theme: 'light',
    density: 'standard',
    children: <ElisyumPreview />,
  },
}

export const DenseOperations: Story = {
  args: {
    theme: 'dark',
    density: 'dense',
    tokens: {
      colors: {
        accent: '#f0b35a',
      },
    },
    children: <ElisyumPreview />,
  },
}
