import { CheckCircle, Info, Warning, X, XCircle } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface NotificationItem {
  id: string
  title: ReactNode
  description?: ReactNode
  tone?: 'info' | 'success' | 'warning' | 'error'
  unread?: boolean
  timestamp?: ReactNode
  action?: ReactNode
}
export interface NotificationStackProps {
  notifications: NotificationItem[]
  onDismiss?: (id: string) => void
  onSelect?: (notification: NotificationItem) => void
  maxVisible?: number
}
const icons = {
  info: Info,
  success: CheckCircle,
  warning: Warning,
  error: XCircle,
}
export function NotificationStack({
  maxVisible = 5,
  notifications,
  onDismiss,
  onSelect,
}: NotificationStackProps) {
  return (
    <section aria-label="Notifications" className="space-y-2">
      {notifications.slice(0, maxVisible).map((notification) => {
        const tone = notification.tone ?? 'info'
        const Icon = icons[tone]
        return (
          <article
            key={notification.id}
            className={`flex gap-3 rounded-lg border border-border bg-surface-card p-4 shadow-sm transition-colors hover:bg-surface-container ${notification.unread ? 'border-primary/40' : ''}`}
          >
            <Icon
              size={20}
              className="mt-0.5 shrink-0 text-primary"
              aria-hidden="true"
            />
            <button
              type="button"
              onClick={() => onSelect?.(notification)}
              className="min-w-0 flex-1 text-start"
            >
              <span className="block font-semibold text-foreground">
                {notification.title}
              </span>
              {notification.description ? (
                <span className="mt-1 block text-body-sm text-muted-foreground">
                  {notification.description}
                </span>
              ) : null}
              <span className="mt-2 block text-label-caps text-muted-foreground">
                {notification.timestamp}
              </span>
              {notification.action ? (
                <span className="mt-3 block">{notification.action}</span>
              ) : null}
            </button>
            <button
              type="button"
              onClick={() => onDismiss?.(notification.id)}
              aria-label="Dismiss notification"
              className="shrink-0 text-muted-foreground hover:text-foreground"
            >
              <X size={16} />
            </button>
          </article>
        )
      })}
    </section>
  )
}
