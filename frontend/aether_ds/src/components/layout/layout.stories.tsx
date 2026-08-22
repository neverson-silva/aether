import type { Meta, StoryObj } from '@storybook/react'
import {
  Bleed,
  Box,
  Container,
  Divider,
  Grid,
  Inline,
  Stack,
  VisuallyHidden,
} from './layout'

const meta = {
  title: 'Foundations/Layout',
  component: Stack,
  tags: ['autodocs'],
} satisfies Meta<typeof Stack>
export default meta
type Story = StoryObj<typeof meta>
const Panel = ({ children }: { children: React.ReactNode }) => (
  <div className="rounded-lg border bg-surface-card p-4">{children}</div>
)
export const StackLayout: Story = {
  args: {
    children: (
      <>
        <Panel>Service health</Panel>
        <Panel>Recent deployments</Panel>
      </>
    ),
  },
}
export const InlineLayout: Story = {
  render: () => (
    <Inline>
      <Panel>Production</Panel>
      <Panel>Healthy</Panel>
      <Panel>3 deployments</Panel>
    </Inline>
  ),
}
export const GridLayout: Story = {
  render: () => (
    <Grid columns="three">
      <Panel>CPU</Panel>
      <Panel>Memory</Panel>
      <Panel>Requests</Panel>
    </Grid>
  ),
}
export const ContainerLayout: Story = {
  render: () => (
    <Container>
      <Stack>
        <Box>Container content</Box>
        <Divider />
        <Box>Bounded to the Aether reading width.</Box>
      </Stack>
    </Container>
  ),
}
export const BleedLayout: Story = {
  render: () => (
    <Container>
      <Panel>
        <Bleed className="bg-surface-container p-4">
          Full-width section inside container
        </Bleed>
      </Panel>
    </Container>
  ),
}
export const HiddenContent: Story = {
  render: () => (
    <>
      <VisuallyHidden>Screen reader context</VisuallyHidden>
      <span aria-hidden="true">Visual content</span>
    </>
  ),
}
