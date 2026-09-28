import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'
import { ScrollArea } from './scroll-area'

export interface LogLine {
  id: string
  timestamp?: ReactNode
  level?: ReactNode
  message: ReactNode
}
export interface LogViewerProps {
  lines: LogLine[]
  follow?: boolean
  className?: string
}

export function LogViewer({ className = '', follow = false, lines }: LogViewerProps) {
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const latestLineId = lines[lines.length - 1]?.id

  useEffect(() => {
    if (!follow || !viewportRef.current) return
    viewportRef.current.scrollTop = viewportRef.current.scrollHeight
  }, [follow, latestLineId, lines.length])

  return (
    <div
      className={`flex max-h-[min(44rem,calc(100dvh-22rem))] min-h-0 flex-col overflow-hidden rounded-xl border border-border-subtle bg-code-canvas ${className}`}
    >
      <ScrollArea
        aria-label="Logs"
        aria-live={follow ? 'polite' : undefined}
        className="min-h-0 flex-1 cursor-text overflow-y-auto px-3 py-2 font-technical text-log"
        role="log"
        ref={viewportRef}
      >
        {lines.length ? (
          lines.map((line, index) => (
            <div
              className="flex min-w-0 items-start gap-3 border-l-2 border-transparent px-2 py-0.5 leading-5 transition-colors duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:border-info hover:bg-surface-1"
              key={line.id}
            >
              <span className="w-10 shrink-0 select-none text-right text-text-subtle">
                {index + 1}
              </span>
              {line.timestamp ? (
                <span className="shrink-0 text-text-subtle">{line.timestamp}</span>
              ) : null}
              {line.level ? <span className="shrink-0 text-info">{line.level}</span> : null}
              <span className="min-w-0 whitespace-pre-wrap break-words text-text-secondary">
                {line.message}
              </span>
            </div>
          ))
        ) : (
          <p className="px-2 py-8 text-center text-supporting text-text-tertiary">
            No log output yet.
          </p>
        )}
      </ScrollArea>
      {follow ? (
        <div className="shrink-0 border-t border-border-subtle px-4 py-2 font-technical text-log text-success">
          Following live output
        </div>
      ) : null}
    </div>
  )
}
