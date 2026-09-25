import { Popover, showToast, type ToastTone } from '@aether/elisyum-ds'
import {
  Bell,
  BellRinging,
  CheckCircle,
  Clock,
  HardDrives,
  Info,
  RocketLaunch,
  Warning,
  XCircle,
} from '@phosphor-icons/react'
import { useNavigate } from '@tanstack/react-router'
import { type ReactNode, useEffect } from 'react'
import { isPublicRoute } from '../api/client'
import type { NotificationItem } from '../hooks/types'
import { useNotificationsStore } from '../stores/notifications'
import { useRealtimeEvent } from './RealtimeProvider'

function isNotifiable(type: string) {
  if (type.startsWith('deploy.'))
    return [
      'deploy.queued',
      'deploy.starting',
      'deploy.failed',
      'deploy.ready',
      'deploy.rolled_back',
      'deploy.cancelled',
    ].includes(type)
  return (
    type.startsWith('backup.') ||
    type.startsWith('server.') ||
    type.startsWith('database.') ||
    type.startsWith('domain.') ||
    type.startsWith('alert.') ||
    type.startsWith('member.') ||
    type.startsWith('env.')
  )
}

function toastToneFor(type: string): ToastTone {
  if (type.includes('failed') || type.includes('error')) return 'error'
  if (type.includes('ready') || type.includes('completed') || type.includes('success'))
    return 'success'
  if (
    type.includes('queued') ||
    type.includes('building') ||
    type.includes('starting') ||
    type.includes('healthcheck')
  )
    return 'warning'
  return 'info'
}

function toItem(event: {
  id: string
  org_id?: string
  type: string
  message?: string
  payload?: unknown
  ts: string
}): NotificationItem {
  return {
    id: event.id,
    org_id: event.org_id || '',
    type: event.type,
    message: event.message || event.type,
    payload: JSON.stringify(event.payload ?? {}),
    read: false,
    created_at: event.ts,
  }
}

function visualFor(type: string) {
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
  if (type.startsWith('alert.'))
    return { Icon: BellRinging, tone: 'text-danger-strong' }
  if (type.startsWith('info.')) return { Icon: Info, tone: 'text-action-strong' }
  return { Icon: Bell, tone: 'text-text-tertiary' }
}

function relativeTime(value: string) {
  const seconds = Math.max(
    0,
    Math.floor((Date.now() - new Date(value).getTime()) / 1000),
  )
  if (seconds < 10) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

const notificationPreviewLimit = 6

export function useNotifications() {
  const unread = useNotificationsStore((state) => state.unread)
  const open = useNotificationsStore((state) => state.bellOpen)
  const toggle = useNotificationsStore((state) => state.toggleBell)
  return { unread, open, toggle }
}

export function NotificationProvider({ children }: { children: ReactNode }) {
  useRealtimeEvent((event, replay) => {
    if (!isNotifiable(event.type)) return
    const item = toItem(event)
    if (replay) useNotificationsStore.getState().prependReplay(item)
    else {
      useNotificationsStore.getState().prepend(item)
      if (event.message && event.type !== 'deploy.build.log')
        showToast(event.message, toastToneFor(event.type))
    }
  })

  useEffect(() => {
    const refresh = () => {
      if (!isPublicRoute()) void useNotificationsStore.getState().refresh()
    }
    refresh()
    window.addEventListener('aether:org', refresh)
    window.addEventListener('aether:auth', refresh)
    return () => {
      window.removeEventListener('aether:org', refresh)
      window.removeEventListener('aether:auth', refresh)
    }
  }, [])

  return <>{children}</>
}

export function NotificationBell() {
  const unread = useNotificationsStore((state) => state.unread)
  const open = useNotificationsStore((state) => state.bellOpen)
  const toggleBell = useNotificationsStore((state) => state.toggleBell)
  const closeBell = useNotificationsStore((state) => state.closeBell)
  return (
    <Popover
      placement="anchor-overlap-center"
      className="!w-[min(380px,calc(100vw-24px))] !min-w-0 !rounded-xl !border-border-subtle/80 !bg-surface-1 !p-0 !shadow-elevation-4 !z-[var(--ely-z-dialog)]"
      onOpenChange={(nextOpen) => {
        if (nextOpen !== open) toggleBell()
      }}
      open={open}
      collisionPadding={12}
      trigger={
        <span className="relative inline-flex size-10 items-center justify-center">
          <Bell size={19} />
          {unread ? (
            <span
              aria-label={`${unread} unread notifications`}
              className="absolute right-0.5 top-0.5 min-w-1.5 rounded-full bg-danger px-0.5 text-center font-technical text-[0.5rem] leading-3 text-text-primary"
            >
              {unread > 99 ? '99+' : unread}
            </span>
          ) : null}
        </span>
      }
      triggerClassName={`inline-flex size-10 cursor-pointer items-center justify-center rounded-xl p-0 text-text-secondary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 hover:text-text-primary active:bg-surface-3 focus-visible:outline-2 focus-visible:outline-focus ${open ? 'bg-surface-2 text-text-primary' : 'bg-transparent'}`}
    >
      <NotificationPanel closeBell={closeBell} />
    </Popover>
  )
}

function NotificationPanel({ closeBell }: { closeBell: () => void }) {
  const navigate = useNavigate()
  const list = useNotificationsStore((state) => state.list)
  const unread = useNotificationsStore((state) => state.unread)
  const markRead = useNotificationsStore((state) => state.markRead)
  const markAllRead = useNotificationsStore((state) => state.markAllRead)
  return (
    <div className="grid text-text-primary">
      <div className="flex items-center justify-between gap-4 px-4 pb-2 pt-4">
        <span className="text-[0.9375rem] font-semibold text-text-primary">
          Notifications
        </span>
        {unread ? (
          <button
            className="text-label text-action-strong transition-colors hover:text-text-primary"
            onClick={() => {
              void markAllRead()
            }}
            type="button"
          >
            Mark all read
          </button>
        ) : null}
      </div>
      {list.length ? (
        <div className="max-h-[min(32rem,calc(100vh-10rem))] overflow-y-auto px-2 pb-2">
          {list.slice(0, notificationPreviewLimit).map((item) => (
            <NotificationRow
              item={item}
              key={item.id}
              onRead={() => {
                if (!item.read) void markRead(item.id)
              }}
            />
          ))}
        </div>
      ) : (
        <div className="grid justify-items-center px-6 pb-8 pt-7 text-center">
          <span className="relative mb-4 grid size-11 place-items-center rounded-full border border-border-subtle bg-surface-1 text-action-strong">
            <span className="absolute size-2 rounded-full bg-action" />
            <span className="absolute size-7 rounded-full border border-action/35" />
          </span>
          <strong className="text-[0.9375rem] font-semibold text-text-primary">
            You&apos;re all caught up
          </strong>
          <span className="mt-1 max-w-56 text-supporting text-text-tertiary">
            Operational events will appear here as they arrive.
          </span>
          <button
            className="mt-5 inline-flex items-center gap-2 text-label text-action-strong transition-colors hover:text-text-primary"
            onClick={() => {
              closeBell()
              void navigate({ to: '/notifications' })
            }}
            type="button"
          >
            View history <span aria-hidden="true">→</span>
          </button>
        </div>
      )}
      {list.length ? (
        <button
          className="px-4 pb-4 pt-2 text-left text-label text-action-strong transition-colors hover:text-text-primary"
          onClick={() => {
            closeBell()
            void navigate({ to: '/notifications' })
          }}
          type="button"
        >
          View all notifications <span aria-hidden="true">→</span>
        </button>
      ) : null}
    </div>
  )
}

function NotificationRow({
  item,
  onRead,
}: {
  item: NotificationItem
  onRead: () => void
}) {
  const navigate = useNavigate()
  const visual = visualFor(item.type)
  const Icon = visual.Icon
  const target = (() => {
    try {
      const payload = JSON.parse(item.payload || '{}') as {
        app_id?: string
        service_id?: string
      }
      return payload.app_id || payload.service_id
    } catch {}
  })()
  return (
    <button
      className={`relative flex w-full items-start gap-3 rounded-[0.6875rem] px-3 py-2.5 text-left transition-[background-color,color] duration-[var(--ely-duration-fast)] hover:bg-surface-2 ${item.read ? '' : 'bg-surface-2/65 before:absolute before:inset-y-2.5 before:left-0 before:w-0.5 before:rounded-full before:bg-action'}`}
      onClick={() => {
        onRead()
        if (target) void navigate({ to: `/apps/${target}` })
      }}
      type="button"
    >
      <Icon
        aria-hidden="true"
        className={`mt-0.5 shrink-0 ${visual.tone}`}
        size={18}
      />
      <span className="min-w-0 flex-1">
        <span
          className={`block break-words text-supporting ${item.read ? 'text-text-secondary' : 'font-medium text-text-primary'}`}
        >
          {item.message}
        </span>
        <span className="mt-1 block font-technical text-log text-text-subtle">
          {item.type} · {relativeTime(item.created_at)}
        </span>
      </span>
      {!item.read ? (
        <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-danger" />
      ) : null}
    </button>
  )
}
