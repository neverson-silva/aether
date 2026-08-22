import type { Meta, StoryObj } from '@storybook/react'
import { DeploymentComposer } from './deployment-composer'

const meta = {
  title: 'Patterns/Deployment Composer',
  component: DeploymentComposer,
  tags: ['autodocs'],
} satisfies Meta<typeof DeploymentComposer>
export default meta
type Story = StoryObj<typeof meta>
export const Review: Story = {
  args: {
    dirty: true,
    source: <p>main at abc123</p>,
    environment: <p>Production</p>,
    variables: <p>3 variables configured</p>,
    review: <p>Ready to deploy.</p>,
  },
}
