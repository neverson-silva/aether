import { CaretDown, CaretRight, Sparkle } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface ReleaseNote {
  id: string
  version: string
  date: ReactNode
  title: string
  summary?: ReactNode
  category?: 'feature' | 'improvement' | 'fix' | 'breaking'
  impact?: 'low' | 'medium' | 'high'
  migration?: ReactNode
  details?: ReactNode
  unread?: boolean
}
export interface ChangelogProps {
  releases: ReleaseNote[]
  onRead?: (id: string) => void
  empty?: ReactNode
}
export function Changelog({
  empty = 'No releases yet.',
  onRead,
  releases,
}: ChangelogProps) {
  const [expanded, setExpanded] = useState<string | null>(null)
  return releases.length ? (
    <section className="space-y-4" aria-label="Changelog">
      {releases.map((release) => {
        const open = expanded === release.id
        return (
          <article
            key={release.id}
            className={`rounded-lg border border-border bg-surface-card p-5 ${release.unread ? 'border-primary/40' : ''}`}
          >
            <button
              type="button"
              onClick={() => {
                setExpanded(open ? null : release.id)
                if (!open) onRead?.(release.id)
              }}
              className="flex w-full items-start gap-3 text-start"
            >
              <span className="mt-0.5 text-primary">
                {open ? <CaretDown size={18} /> : <CaretRight size={18} />}
              </span>
              <span className="min-w-0 flex-1">
                <span className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-body-sm text-primary">
                    {release.version}
                  </span>
                  {release.unread ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-label-caps text-primary">
                      <Sparkle size={12} />
                      New
                    </span>
                  ) : null}
                  <span className="text-label-caps text-muted-foreground">
                    {release.category ?? 'release'}
                  </span>
                </span>
                <span className="mt-2 block text-body-md font-semibold text-foreground">
                  {release.title}
                </span>
                {release.summary ? (
                  <span className="mt-1 block text-body-sm text-muted-foreground">
                    {release.summary}
                  </span>
                ) : null}
                <time className="mt-2 block text-label-caps text-muted-foreground">
                  {release.date}
                </time>
              </span>
            </button>
            {open ? (
              <div className="ml-8 mt-4 space-y-3 border-t border-border pt-4 text-body-sm text-foreground">
                {release.details ? <div>{release.details}</div> : null}
                {release.migration ? (
                  <div className="rounded-md bg-surface-container p-3">
                    <strong className="block text-body-sm">
                      Migration note
                    </strong>
                    {release.migration}
                  </div>
                ) : null}
              </div>
            ) : null}
          </article>
        )
      })}
    </section>
  ) : (
    <div className="rounded-lg border border-dashed border-border p-8 text-center text-body-sm text-muted-foreground">
      {empty}
    </div>
  )
}
