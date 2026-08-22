import type { Meta, StoryObj } from '@storybook/react'
import { AsyncSearchInput } from './async-search-input'

const meta = {
  title: 'Forms/Async Search Input',
  component: AsyncSearchInput,
  tags: ['autodocs'],
} satisfies Meta<typeof AsyncSearchInput>
export default meta
type Story = StoryObj<typeof meta>

export const RemoteResources: Story = {
  args: {
    label: 'Service',
    description:
      'Results are fetched as you type; the full catalog is not loaded.',
    placeholder: 'Search services',
    loadOptions: async (query) => {
      await new Promise((resolve) => window.setTimeout(resolve, 500))
      return ['aether-api', 'aether-web', 'aether-worker', 'billing-service']
        .filter((item) => item.includes(query.toLowerCase()))
        .map((item) => ({
          value: item,
          label: item,
          description: 'Production resource',
        }))
    },
  },
}
