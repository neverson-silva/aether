import type { ReactNode } from 'react'
import { ActivityFeed, type ActivityItem } from './activity-feed'
import { Marker } from './marker'

export interface RealtimeActivitySurfaceProps {
  items: ActivityItem[]
  connected?: boolean
  stale?: boolean
  title?: ReactNode
  className?: string
}
export function RealtimeActivitySurface({
  className = '',
  connected = true,
  items,
  stale = false,
  title = 'Live activity',
}: RealtimeActivitySurfaceProps) {
  const state = connected ? (stale ? 'Stale' : 'Live') : 'Disconnected'
  const tone = connected && !stale ? 'success' : 'warning'
  return (
    <section
      aria-live="polite"
      className={`grid gap-4 border-y border-border-subtle bg-surface-1 py-4 ${className}`}
    >
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-section-title text-text-primary">{title}</h2>
        <span className="inline-flex items-center gap-2 text-log text-text-secondary">
          <Marker
            aria-hidden="true"
            tone={tone}
          />
          {state}
        </span>
      </div>
      <ActivityFeed items={items} />
    </section>
  )
}
