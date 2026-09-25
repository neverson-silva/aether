import type { ReactNode } from 'react'
import { Marker } from './marker'
import type { BadgeTone } from './badge'

export interface TimelineItem {
  id: string
  title: ReactNode
  description?: ReactNode
  timestamp?: ReactNode
  status?: ReactNode
  tone?: BadgeTone
}
export interface TimelineProps {
  items: TimelineItem[]
  className?: string
}

export function Timeline({ className = '', items }: TimelineProps) {
  return (
    <ol
      aria-label="Timeline"
      className={`grid gap-0 ${className}`}
    >
      {items.map((item, index) => (
        <li
          className="relative flex gap-3 pb-6 last:pb-0"
          key={item.id}
        >
          <div
            aria-hidden="true"
            className="relative flex w-4 justify-center"
          >
            <Marker
              className="z-10 mt-1"
              tone={item.tone ?? 'accent'}
            />
            {index < items.length - 1 ? (
              <span className="absolute top-3 h-full w-px bg-border-default" />
            ) : null}
          </div>
          <div className="grid min-w-0 flex-1 gap-1">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-body text-text-primary">{item.title}</span>
              {item.status ? (
                <span className="inline-flex items-center gap-2 text-log text-text-secondary">
                  <Marker
                    aria-hidden="true"
                    tone={item.tone ?? 'neutral'}
                  />
                  {item.status}
                </span>
              ) : null}
            </div>
            {item.description ? (
              <div className="text-supporting text-text-tertiary">
                {item.description}
              </div>
            ) : null}
            {item.timestamp ? (
              <time className="font-technical text-log text-text-subtle">
                {item.timestamp}
              </time>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  )
}
