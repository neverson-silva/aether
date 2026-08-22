import {
  Bell,
  CheckCircle,
  GitBranch,
  Rocket,
  Warning,
} from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface ActivityItem {
  id: string
  title: ReactNode
  description?: ReactNode
  timestamp: ReactNode
  actor?: ReactNode
  type?: 'deployment' | 'change' | 'warning' | 'success' | 'info'
  unread?: boolean
  group?: string
}
export interface ActivityFeedProps {
  items: ActivityItem[]
  loading?: boolean
  empty?: ReactNode
  realtime?: boolean
  onLoadMore?: () => void
  hasMore?: boolean
  onItemClick?: (item: ActivityItem) => void
}
export function ActivityFeed({
  empty = 'No activity yet.',
  hasMore,
  items,
  loading,
  onItemClick,
  onLoadMore,
  realtime,
}: ActivityFeedProps) {
  const icons = {
    deployment: Rocket,
    change: GitBranch,
    warning: Warning,
    success: CheckCircle,
    info: Bell,
  }
  return (
    <section aria-label="Activity feed" className="space-y-2">
      {realtime ? (
        <div className="flex items-center gap-2 text-body-sm text-status-success">
          <span className="size-2 animate-pulse rounded-full bg-status-success" />
          Live activity
        </div>
      ) : null}
      {loading ? (
        <div className="space-y-2">
          <div className="h-16 animate-pulse rounded-lg bg-surface-container" />
          <div className="h-16 animate-pulse rounded-lg bg-surface-container" />
        </div>
      ) : items.length ? (
        <div className="space-y-1">
          {items.map((item) => {
            const Icon = icons[item.type ?? 'info']
            return (
              <button
                type="button"
                key={item.id}
                onClick={() => onItemClick?.(item)}
                className={`flex w-full gap-3 rounded-lg p-3 text-start transition-colors hover:bg-surface-container ${item.unread ? 'bg-primary/5' : ''}`}
              >
                <span className="mt-0.5 inline-flex size-8 shrink-0 items-center justify-center rounded-full bg-surface-container text-primary">
                  <Icon size={16} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="text-body-sm font-semibold text-foreground">
                      {item.title}
                    </span>
                    <time className="text-body-sm text-muted-foreground">
                      {item.timestamp}
                    </time>
                  </span>
                  {item.description ? (
                    <span className="mt-1 block text-body-sm text-muted-foreground">
                      {item.description}
                    </span>
                  ) : null}
                  {item.actor ? (
                    <span className="mt-2 block text-label-caps text-muted-foreground">
                      {item.actor}
                    </span>
                  ) : null}
                </span>
                {item.unread ? (
                  <span className="mt-2 size-2 shrink-0 rounded-full bg-primary" />
                ) : null}
              </button>
            )
          })}
        </div>
      ) : (
        <div className="rounded-lg border border-dashed border-border p-8 text-center text-body-sm text-muted-foreground">
          {empty}
        </div>
      )}
      {hasMore ? (
        <button
          type="button"
          onClick={onLoadMore}
          className="w-full rounded-md border border-border px-3 py-2 text-body-sm text-muted-foreground hover:bg-surface-container"
        >
          Load more
        </button>
      ) : null}
    </section>
  )
}
