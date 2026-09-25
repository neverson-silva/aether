import type { ReactNode } from 'react'
import { Timeline, type TimelineItem } from './timeline'

export interface ReleaseNote extends TimelineItem {
  version: ReactNode
}
export interface ChangelogProps {
  releases: ReleaseNote[]
  className?: string
}

export function Changelog({ className = '', releases }: ChangelogProps) {
  return (
    <div className={`grid gap-4 ${className}`}>
      <h2 className="text-section-title text-text-primary">Changelog</h2>
      <Timeline
        items={releases.map((release) => ({
          ...release,
          title: (
            <span className="flex gap-2">
              <strong>{release.version}</strong>
              {release.title}
            </span>
          ),
        }))}
      />
    </div>
  )
}
