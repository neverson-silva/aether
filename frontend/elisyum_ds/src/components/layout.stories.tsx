import type { Meta, StoryObj } from '@storybook/react-vite'
import { Card } from './card'
import { Grid, Inline, Stack } from './layout'

const meta = {
  component: Stack,
  title: 'Foundations/Layout',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Stack>
export default meta
type Story = StoryObj<typeof meta>
export const Composition: Story = {
  args: {
    children: (
      <Grid columns={2}>
        <Card>Service one</Card>
        <Card>Service two</Card>
        <Inline>
          <span className="text-body">Status</span>
          <span className="text-supporting">Healthy</span>
        </Inline>
      </Grid>
    ),
  },
}
