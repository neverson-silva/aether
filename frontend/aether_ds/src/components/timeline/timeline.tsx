import { Check, Circle, Warning } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface TimelineEvent {
  id: string
  title: ReactNode
  description?: ReactNode
  timestamp: ReactNode
  status?: 'complete' | 'active' | 'warning' | 'error' | 'pending'
  actor?: ReactNode
  details?: ReactNode
}
export interface TimelineProps {
  events: TimelineEvent[]
  empty?: ReactNode
  realtime?: boolean
}
export function Timeline({
  empty = 'No events yet.',
  events,
  realtime,
}: TimelineProps) {
  const icon = (status: TimelineEvent['status']) =>
    status === 'complete' ? (
      <Check size={14} />
    ) : status === 'warning' || status === 'error' ? (
      <Warning size={14} />
    ) : (
      <Circle size={10} weight="fill" />
    )
  return events.length ? (
    <ol className="space-y-0">
      {events.map((event, index) => (
        <li key={event.id} className="relative flex gap-3 pb-6 last:pb-0">
          <div className="relative flex w-5 shrink-0 justify-center">
            <span
              className={`z-10 inline-flex size-5 items-center justify-center rounded-full border ${event.status === 'error' ? 'border-status-danger bg-status-danger text-status-danger-foreground' : event.status === 'warning' ? 'border-status-warning bg-status-warning text-status-warning-foreground' : event.status === 'complete' ? 'border-status-success bg-status-success text-status-success-foreground' : 'border-primary bg-primary/15 text-primary'} ${event.status === 'active' && realtime ? 'animate-pulse' : ''}`}
            >
              {icon(event.status)}
            </span>
            {index < events.length - 1 ? (
              <span className="absolute top-5 h-full w-px bg-border" />
            ) : null}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <strong className="text-body-sm text-foreground">
                {event.title}
              </strong>
              <time className="text-body-sm text-muted-foreground">
                {event.timestamp}
              </time>
            </div>
            {event.description ? (
              <div className="mt-1 text-body-sm text-muted-foreground">
                {event.description}
              </div>
            ) : null}
            {event.actor ? (
              <div className="mt-2 text-label-caps text-muted-foreground">
                {event.actor}
              </div>
            ) : null}
            {event.details ? (
              <div className="mt-3 rounded-md bg-surface-container p-3 text-body-sm">
                {event.details}
              </div>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  ) : (
    <div className="rounded-lg border border-dashed border-border p-8 text-center text-body-sm text-muted-foreground">
      {empty}
    </div>
  )
}
