import {
  Check,
  CloudArrowUp,
  Copy,
  DotsThree,
  Info,
  MagnifyingGlass,
  Warning,
  X,
} from '@phosphor-icons/react'
import type { Meta, StoryObj } from '@storybook/react'
import { Alert } from '../alert/alert'
import { Avatar } from '../avatar/avatar'
import { Badge } from '../badge/badge'
import { ButtonGroup } from '../button-group/button-group'
import { EmptyState } from '../empty-state/empty-state'
import { IconButton } from '../icon-button/icon-button'
import { Kbd } from '../kbd/kbd'
import { Label } from '../label/label'
import { Link } from '../link/link'
import { Progress } from '../progress/progress'
import { Separator } from '../separator/separator'
import { Skeleton } from '../skeleton/skeleton'
import { Spinner } from '../spinner/spinner'

const meta = {
  title: 'Foundations/Primitives',
  component: Badge,
  tags: ['autodocs'],
} satisfies Meta<typeof Badge>
export default meta
type Story = StoryObj<typeof meta>

export const LinkStates: Story = {
  render: () => (
    <div className="flex gap-4">
      <Link href="#">Default link</Link>
      <Link href="#" tone="muted" underline>
        Muted link
      </Link>
      <Link href="#" external>
        External link
      </Link>
      <Link href="#" disabled>
        Disabled link
      </Link>
    </div>
  ),
}
export const ButtonGroupStates: Story = {
  render: () => (
    <div className="space-y-4">
      <ButtonGroup>
        <button type="button" className="rounded-md border px-3 py-2">
          Deploy
        </button>
        <button type="button" className="rounded-md border px-3 py-2">
          Rollback
        </button>
      </ButtonGroup>
      <ButtonGroup orientation="vertical" attached>
        <button type="button" className="rounded-md border px-3 py-2">
          Start
        </button>
        <button type="button" className="rounded-md border px-3 py-2">
          Stop
        </button>
      </ButtonGroup>
    </div>
  ),
}
export const IconButtonStates: Story = {
  render: () => (
    <div className="flex gap-2">
      <IconButton icon={MagnifyingGlass} label="Search" />
      <IconButton icon={DotsThree} label="More actions" pressed />
      <IconButton icon={Copy} label="Copy" loading />
      <IconButton icon={X} label="Disabled" disabled />
    </div>
  ),
}
export const BadgeStates: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge>Neutral</Badge>
      <Badge tone="info" icon={Info}>
        Info
      </Badge>
      <Badge tone="success" icon={Check}>
        Healthy
      </Badge>
      <Badge tone="warning" icon={Warning}>
        Degraded
      </Badge>
      <Badge tone="danger" dot live>
        Failed
      </Badge>
      <Badge onRemove={() => undefined}>Removable</Badge>
    </div>
  ),
}
export const AvatarStates: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <Avatar fallback="AS" status="online" />
      <Avatar fallback="DS" size="sm" status="away" />
      <Avatar fallback="ER" size="lg" status="offline" />
      <Avatar src="/missing-avatar.png" fallback="FB" />
    </div>
  ),
}
export const KeyboardShortcut: Story = {
  render: () => (
    <div className="flex items-center gap-2 text-body-sm">
      Open command palette <Kbd keys={['⌘', 'K']} />
    </div>
  ),
}
export const Labels: Story = {
  render: () => (
    <div className="space-y-3">
      <Label htmlFor="service">
        Service name <span className="text-muted-foreground">(required)</span>
      </Label>
      <Label htmlFor="region" optional>
        Region
      </Label>
      <Label htmlFor="disabled-field" disabled>
        Disabled field
      </Label>
    </div>
  ),
}
export const Separators: Story = {
  render: () => (
    <div className="space-y-4">
      <div>Before</div>
      <Separator />
      <div>After</div>
      <div className="flex h-8 items-center gap-4">
        <span>Left</span>
        <Separator orientation="vertical" />
        <span>Right</span>
      </div>
    </div>
  ),
}
export const LoadingPrimitives: Story = {
  render: () => (
    <div className="space-y-4">
      <Spinner label="Loading deployment" />
      <Skeleton variant="text" />
      <Skeleton variant="avatar" />
      <Skeleton variant="card" />
    </div>
  ),
}
export const ProgressStates: Story = {
  render: () => (
    <div className="space-y-4">
      <Progress value={62} label="Deployment progress" />
      <Progress value={100} status="success" label="Complete" />
      <Progress indeterminate label="Working" />
      <Progress value={35} status="danger" label="Failed" />
    </div>
  ),
}
export const AlertStates: Story = {
  render: () => (
    <div className="space-y-3">
      <Alert title="Deployment started" icon={CloudArrowUp}>
        Your production deployment is now running.
      </Alert>
      <Alert tone="success" title="Deployment ready" icon={Check}>
        The service is healthy.
      </Alert>
      <Alert tone="warning" title="High latency" icon={Warning} dismissible>
        Response time is above the configured threshold.
      </Alert>
      <Alert tone="danger" title="Deployment failed" icon={X}>
        Check the logs and retry the operation.
      </Alert>
    </div>
  ),
}
export const EmptyStates: Story = {
  render: () => (
    <EmptyState
      icon={MagnifyingGlass}
      title="No services found"
      description="Try changing your search or create a new service."
      action={
        <button
          type="button"
          className="rounded-md bg-primary px-3 py-2 text-primary-foreground"
        >
          Create service
        </button>
      }
    />
  ),
}
export const MotionStates: Story = {
  render: () => (
    <div className="flex gap-4">
      <div className="aether-enter rounded-md border bg-surface-card p-4">
        Enter
      </div>
      <div
        className="aether-shimmer h-12 w-32 rounded-md border"
        aria-hidden="true"
      />
      <div className="aether-exit rounded-md border bg-surface-card p-4">
        Exit
      </div>
    </div>
  ),
}
