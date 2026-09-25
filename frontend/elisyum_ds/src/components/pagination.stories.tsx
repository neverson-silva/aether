import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { Pagination } from './pagination'

const meta = {
  component: Pagination,
  title: 'Data/Pagination',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Pagination>
export default meta
type Story = StoryObj<typeof meta>
export const Results: Story = {
  args: { page: 2, pageCount: 8, onPageChange: fn() },
  render: (args) => {
    const [page, setPage] = useState(args.page)
    return (
      <Pagination
        {...args}
        onPageChange={(nextPage) => {
          args.onPageChange(nextPage)
          setPage(nextPage)
        }}
        page={page}
      />
    )
  },
  play: async ({ args, canvas }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Next page' }))
    await expect(canvas.getByText('03 / 08')).toBeVisible()
    await expect(args.onPageChange).toHaveBeenCalledWith(3)
    await userEvent.click(canvas.getByRole('button', { name: 'Previous page' }))
    await expect(canvas.getByText('02 / 08')).toBeVisible()
  },
}
