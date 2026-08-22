import {
  ActivityIcon as Activity,
  Gear,
  Package,
  Rocket,
} from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react'
import { Accordion } from '../accordion/accordion'
import { AppHeader } from '../app-header/app-header'
import { Breadcrumb } from '../breadcrumb/breadcrumb'
import { Card } from '../card/card'
import { Collapsible } from '../collapsible/collapsible'
import { Item, ItemGroup } from '../item/item'
import { Pagination } from '../pagination/pagination'
import { Resizable } from '../resizable/resizable'
import { ScrollArea } from '../scroll-area/scroll-area'
import { SheetSidebar } from '../sheet-sidebar/sheet-sidebar'
import { Sidebar } from '../sidebar/sidebar'
import { Tabs } from '../tabs/tabs'

const meta = {
  title: 'Patterns/Application Structure',
  component: Card,
  tags: ['autodocs'],
} satisfies Meta<typeof Card>
export default meta
type Story = StoryObj<typeof meta>
const items = [
  { label: 'Overview', icon: <Activity size={18} />, active: true, href: '#' },
  {
    label: 'Deployments',
    icon: <Rocket size={18} />,
    badge: <span className="text-xs">3</span>,
    href: '#',
  },
  { label: 'Settings', icon: <Gear size={18} />, href: '#' },
]
export const CardsAndItems: Story = {
  render: () => (
    <div className="space-y-4">
      <Card variant="metric" header="Requests" footer="Last 24 hours">
        <div className="text-display-lg">24.8k</div>
      </Card>
      <ItemGroup>
        <Item
          title="Aether API"
          description="Production service"
          media={<Package size={24} />}
          metadata="Healthy"
          actions={
            <button type="button" className="rounded border px-2 py-1">
              Open
            </button>
          }
          interactive
        />
        <Item
          title="Aether Worker"
          description="Staging service"
          selected
          media={<Package size={24} />}
        />
      </ItemGroup>
    </div>
  ),
}
export const Navigation: Story = {
  render: () => (
    <div className="flex h-96">
      <Sidebar
        items={items}
        header={<span>AETHER</span>}
        footer={<span className="text-xs text-muted-foreground">v0.1.0</span>}
      />
      <div className="flex-1">
        <AppHeader
          workspace="Aether Platform"
          breadcrumb={[
            { label: 'Projects', href: '#' },
            { label: 'Aether API', current: true },
          ]}
          environment={
            <button
              type="button"
              className="rounded border px-2 py-1 text-body-sm"
            >
              Production
            </button>
          }
          user={<span className="text-body-sm">AS</span>}
        />
      </div>
    </div>
  ),
}
export const Breadcrumbs: Story = {
  render: () => (
    <Breadcrumb
      items={[
        { label: 'Projects', href: '#' },
        { label: 'Services', href: '#' },
        { label: 'Aether API', current: true },
      ]}
    />
  ),
}
export const PaginationStates: Story = {
  render: () => (
    <Pagination
      page={3}
      pageCount={12}
      pageSize={25}
      onPageChange={() => undefined}
      onPageSizeChange={() => undefined}
    />
  ),
}
export const TabsStates: Story = {
  render: () => (
    <Tabs
      items={[
        {
          value: 'overview',
          label: 'Overview',
          content: <p>Service overview content.</p>,
        },
        { value: 'logs', label: 'Logs', content: <p>Deployment logs.</p> },
        {
          value: 'settings',
          label: 'Settings',
          content: <p>Service settings.</p>,
          disabled: true,
        },
      ]}
    />
  ),
}
export const Disclosure: Story = {
  render: () => (
    <div className="space-y-4">
      <Accordion
        items={[
          {
            value: 'health',
            title: 'Health checks',
            content: 'All production health checks are passing.',
          },
          {
            value: 'deploy',
            title: 'Latest deployment',
            content: 'Deployed from main branch 4 minutes ago.',
          },
        ]}
      />
      <Collapsible title="Advanced details">
        <p>Request IDs, runtime metadata and infrastructure details.</p>
      </Collapsible>
    </div>
  ),
}
export const ResizablePanels: Story = {
  render: () => (
    <div className="h-64">
      <Resizable sidebar={<div className="p-4">Resource tree</div>}>
        <div className="p-4">Logs and details</div>
      </Resizable>
    </div>
  ),
}
export const Scrollable: Story = {
  render: () => (
    <ScrollArea>
      <div className="space-y-2 p-4">
        {Array.from({ length: 20 }, (_, index) => (
          <div key={index} className="rounded border border-border p-3">
            Deployment event {index + 1}
          </div>
        ))}
      </div>
    </ScrollArea>
  ),
}
export const MobileSheet: Story = {
  render: () => (
    <SheetSidebar
      trigger={
        <button
          type="button"
          className="rounded-md border border-border px-3 py-2"
        >
          Open navigation
        </button>
      }
      title="Navigation"
      description="Choose an area of the platform."
    >
      <div className="space-y-2">
        <a
          href="/overview"
          className="block rounded p-2 hover:bg-surface-container"
        >
          Overview
        </a>
        <a
          href="/deployments"
          className="block rounded p-2 hover:bg-surface-container"
        >
          Deployments
        </a>
      </div>
    </SheetSidebar>
  ),
}
