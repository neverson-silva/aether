import {
  Button,
  Card,
  EmptyState,
  Marker,
  Skeleton,
  Typography,
} from '@aether/elisyum-ds'
import {
  Bell,
  CheckCircle,
  Clock,
  HardDrives,
  Info,
  RocketLaunch,
  Warning,
  XCircle,
} from '@phosphor-icons/react'
import { useQueryClient } from '@tanstack/react-query'
import { useRealtimeEvent } from '../components/RealtimeProvider'
import type { NotificationItem } from '../hooks/types'
import { useMarkAllRead } from '../hooks/use-mark-all-read'
import { useMarkRead } from '../hooks/use-mark-read'
import { useNotifications } from '../hooks/use-notifications'

export function NotificationsField() {
  const queryClient = useQueryClient()
  const query = useNotifications()
  const markRead = useMarkRead()
  const markAllRead = useMarkAllRead()
  useRealtimeEvent((event) => {
    if (
      event.type.startsWith('deploy.') ||
      event.type.startsWith('backup.') ||
      event.type.startsWith('server.') ||
      event.type.startsWith('database.') ||
      event.type.startsWith('domain.') ||
      event.type.startsWith('alert.') ||
      event.type.startsWith('member.') ||
      event.type.startsWith('env.')
    )
      void queryClient.invalidateQueries({ queryKey: ['notifications'] })
  })
  const events = query.data ?? []
  const unread = events.filter((event) => !event.read).length
  return (
    <div className="grid gap-7">
      <header className="flex flex-wrap items-end justify-between gap-5">
        <div className="grid gap-3">
          <div className="flex items-center gap-3 text-label tracking-[0.16em] text-text-subtle">
            <Bell size={17} />
            SIGNALS / INBOX
          </div>
          <Typography
            as="h1"
            role="page-title"
          >
            Operational signals
          </Typography>
          <Typography role="supporting">
            Acknowledge changes, failures and recovery events without losing the
            organization context.
          </Typography>
        </div>
        {unread ? (
          <Button
            disabled={markAllRead.isPending}
            onClick={() => markAllRead.mutate()}
            tone="ghost"
          >
            Mark all as read
          </Button>
        ) : null}
      </header>
      {query.isError ? (
        <div className="rounded-xl border border-danger/45 bg-danger-soft px-4 py-4 text-supporting text-danger-strong">
          Notifications could not be loaded.{' '}
          <button
            className="ml-2 underline"
            onClick={() => query.refetch()}
            type="button"
          >
            Retry request
          </button>
        </div>
      ) : null}
      {query.isLoading ? (
        <Skeleton className="h-72 rounded-2xl" />
      ) : events.length ? (
        <Card className="grid gap-2 rounded-2xl border-border-subtle bg-surface-1 p-3">
          <div className="flex items-center justify-between gap-4 px-2 pb-2">
            <div className="flex items-center gap-3 text-label tracking-[0.12em] text-text-subtle">
              <Marker tone={unread ? 'accent' : 'success'} />
              SIGNAL HISTORY
            </div>
            <span className="font-technical text-log text-text-tertiary">
              {unread ? `${unread} unread` : 'ALL ACKNOWLEDGED'}
            </span>
          </div>
          {events.slice(0, 100).map((event) => (
            <SignalRow
              event={event}
              key={event.id}
              onRead={() => {
                if (!event.read) markRead.mutate(event.id)
              }}
            />
          ))}
        </Card>
      ) : (
        <EmptyState
          title="You're all caught up"
          description="Operational events will appear here as they arrive through the live event stream."
          icon={<CheckCircle size={28} />}
        />
      )}
    </div>
  )
}

function SignalRow({ event, onRead }: { event: NotificationItem; onRead: () => void }) {
  const visual = signalVisual(event.type)
  const Icon = visual.Icon
  return (
    <button
      aria-label={`${event.read ? 'Read' : 'Unread'} signal: ${event.message || event.type}`}
      className={`relative grid min-h-[4.75rem] w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 rounded-xl border px-4 text-left transition-[background-color,border-color] ${event.read ? 'border-transparent bg-surface-2 hover:border-border-default hover:bg-surface-3' : 'border-action/45 bg-surface-2 before:absolute before:inset-y-3 before:left-0 before:w-0.5 before:rounded-full before:bg-action hover:bg-surface-3'}`}
      onClick={onRead}
      type="button"
    >
      <span
        className={`grid size-9 place-items-center rounded-lg bg-surface-1 ${visual.tone}`}
      >
        <Icon size={18} />
      </span>
      <span className="grid min-w-0 gap-1">
        <span
          className={`break-words text-supporting ${event.read ? 'text-text-secondary' : 'font-medium text-text-primary'}`}
        >
          {event.message || event.type}
        </span>
        <span className="font-technical text-log text-text-subtle">
          {event.type} · {formatTimestamp(event.created_at)}
        </span>
      </span>
      <span
        className={`font-technical text-log ${event.read ? 'text-text-tertiary' : 'text-action-strong'}`}
      >
        {event.read ? 'READ' : 'NEW'}
      </span>
    </button>
  )
}

function signalVisual(type: string) {
  if (type.includes('failed') || type.includes('error'))
    return { Icon: XCircle, tone: 'text-danger-strong' }
  if (
    type.includes('ready') ||
    type.includes('finished') ||
    type.includes('completed') ||
    type.includes('success') ||
    type.includes('recovered')
  )
    return { Icon: CheckCircle, tone: 'text-success-strong' }
  if (type.includes('rolled_back') || type.includes('cancelled'))
    return { Icon: Warning, tone: 'text-warning-strong' }
  if (
    type.includes('queued') ||
    type.includes('building') ||
    type.includes('starting') ||
    type.includes('healthcheck')
  )
    return { Icon: Clock, tone: 'text-warning-strong' }
  if (type.startsWith('deploy.'))
    return { Icon: RocketLaunch, tone: 'text-action-strong' }
  if (
    type.startsWith('server.') ||
    type.startsWith('database.') ||
    type.startsWith('backup.')
  )
    return { Icon: HardDrives, tone: 'text-action-strong' }
  if (type.startsWith('info.')) return { Icon: Info, tone: 'text-action-strong' }
  return { Icon: Bell, tone: 'text-text-tertiary' }
}

function formatTimestamp(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? value
    : date.toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })
}
