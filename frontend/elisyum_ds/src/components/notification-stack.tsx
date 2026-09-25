import { X } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import { IconButton } from './icon-button'
import type { BadgeTone } from './badge'
import { Marker } from './marker'

export interface NotificationItem {
  id: string
  title: ReactNode
  description?: ReactNode
  tone?: BadgeTone
  onDismiss?: () => void
}
export interface NotificationStackProps {
  items: NotificationItem[]
  className?: string
}
function notificationBorder(tone: BadgeTone) {
  switch (tone) {
    case 'danger':
      return 'border-danger'
    case 'warning':
      return 'border-warning'
    case 'success':
      return 'border-success'
    case 'accent':
      return 'border-action'
    default:
      return 'border-border-default'
  }
}
export function NotificationStack({ className = '', items }: NotificationStackProps) {
  return (
    <ul
      aria-label="Notifications"
      aria-live="polite"
      className={`m-0 grid list-none gap-2 p-0 ${className}`}
    >
      {items.map((item) => {
        const tone = item.tone ?? 'neutral'
        return (
          <li
            className={`grid gap-2 rounded-xl border-l-2 ${notificationBorder(tone)} bg-overlay p-4 shadow-elevation-1`}
            key={item.id}
          >
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <Marker
                    aria-hidden="true"
                    tone={tone}
                  />
                  <h3 className="text-label text-text-primary">{item.title}</h3>
                </div>
                {item.description ? (
                  <p className="mt-1 text-supporting text-text-secondary">
                    {item.description}
                  </p>
                ) : null}
              </div>
              {item.onDismiss ? (
                <IconButton
                  label="Dismiss notification"
                  onClick={item.onDismiss}
                  size="sm"
                >
                  <X
                    aria-hidden="true"
                    size={16}
                  />
                </IconButton>
              ) : null}
            </div>
          </li>
        )
      })}
    </ul>
  )
}
