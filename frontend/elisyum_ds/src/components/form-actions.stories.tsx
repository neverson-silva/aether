import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { Button } from './button'
import { FormActions } from './form-actions'

const meta = {
  component: FormActions,
  title: 'Forms/FormActions',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof FormActions>
export default meta
type Story = StoryObj<typeof meta>
export const Submit: Story = {
  args: {
    secondary: <Button tone="ghost">Cancel</Button>,
    primary: <Button type="submit">Save changes</Button>,
  },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('group', { name: 'Form actions' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Save changes' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))
  },
}
