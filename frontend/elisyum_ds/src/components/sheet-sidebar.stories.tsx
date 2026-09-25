import type { Meta, StoryObj } from '@storybook/react-vite'
import { SheetSidebar } from './sheet-sidebar'

const meta = {
  component: SheetSidebar,
  title: 'Overlays/SheetSidebar',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof SheetSidebar>
export default meta
type Story = StoryObj<typeof meta>
export const MobileNavigation: Story = {
  args: {
    trigger: 'Open navigation',
    title: 'Workspace navigation',
    children: (
      <nav className="grid gap-2 text-body text-text-secondary">
        <a href="/applications">Applications</a>
        <a href="/deployments">Deployments</a>
        <a href="/settings">Settings</a>
      </nav>
    ),
  },
}
