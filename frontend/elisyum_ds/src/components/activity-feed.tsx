import type { ReactNode } from 'react'
import { Avatar } from './avatar'
import { Marker } from './marker'

export interface ActivityItem {
  id: string
  actor: string
  action: ReactNode
  timestamp?: ReactNode
  tone?: 'neutral' | 'accent' | 'success' | 'warning' | 'danger'
}
export interface ActivityFeedProps {
  items: ActivityItem[]
  className?: string
}
export function ActivityFeed({ className = '', items }: ActivityFeedProps) {
  return (
    <ul
      aria-label="Activity"
      className={`m-0 grid list-none gap-3 p-0 ${className}`}
    >
      {items.map((item, index) => (
        <li
          className="relative flex gap-3"
          key={item.id}
        >
          <Avatar
            name={item.actor}
            size="sm"
          />
          <div className="grid min-w-0 gap-1">
            <p className="flex items-start gap-2 text-supporting text-text-secondary">
              <Marker
                aria-hidden="true"
                tone={item.tone === 'accent' ? 'accent' : (item.tone ?? 'neutral')}
              />
              <span>
                <strong className="text-text-primary">{item.actor}</strong>{' '}
                {item.action}
              </span>
            </p>
            {item.timestamp ? (
              <time className="text-log text-text-tertiary">{item.timestamp}</time>
            ) : null}
          </div>
          {index < items.length - 1 ? (
            <span
              aria-hidden="true"
              className="absolute bottom-[-0.75rem] left-4 top-8 w-px bg-border-subtle"
            />
          ) : null}
        </li>
      ))}
    </ul>
  )
}
