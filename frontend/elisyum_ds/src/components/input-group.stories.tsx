import { MagnifyingGlass } from '@phosphor-icons/react'
import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { Input } from './input'
import { InputGroup } from './input-group'

const meta = {
  component: InputGroup,
  title: 'Components/InputGroup',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof InputGroup>
export default meta
type Story = StoryObj<typeof meta>
export const Search: Story = {
  args: { children: null },
  render: () => (
    <div className="w-96">
      <InputGroup leading={<MagnifyingGlass size={18} />}>
        <Input
          aria-label="Search resources"
          placeholder="Search resources"
        />
      </InputGroup>
    </div>
  ),
}
export const ReactHookForm: Story = {
  args: { children: null },
  render: () => {
    const form = useForm<{ query: string }>({ defaultValues: { query: '' } })
    return (
      <form
        className="w-96"
        onSubmit={form.handleSubmit(() => undefined)}
      >
        <InputGroup leading={<MagnifyingGlass size={18} />}>
          <Input
            aria-label="Search resources"
            {...form.register('query')}
          />
        </InputGroup>
      </form>
    )
  },
}
