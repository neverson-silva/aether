import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent } from 'storybook/test'
import { CopyButton } from './copy-button'

const meta = {
  component: CopyButton,
  title: 'Components/CopyButton',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof CopyButton>
export default meta
type Story = StoryObj<typeof meta>
export const Identifier: Story = {
  args: { value: 'app_01HZX9', label: 'Copy application identifier' },
  play: async ({ canvas }) => {
    await userEvent.click(
      canvas.getByRole('button', { name: 'Copy application identifier' }),
    )
    await expect(
      canvas.getByRole('button', { name: /Copied|Copy failed/ }),
    ).toBeVisible()
  },
}
