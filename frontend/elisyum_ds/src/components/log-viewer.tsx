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
    <ScrollArea
      aria-label="Logs"
      aria-live={follow ? 'polite' : undefined}
      className={`max-h-[min(44rem,calc(100dvh-22rem))] cursor-text overflow-y-auto rounded-xl border border-border-subtle bg-code-canvas px-4 py-3 ${className}`}
      role="log"
      ref={viewportRef}
    >
      <div className="grid gap-0.5 font-technical text-log">
        {lines.length ? (
          lines.map((line, index) => (
            <div
              className="grid grid-cols-[3rem_auto_minmax(0,1fr)] gap-3 border-l-2 border-transparent px-2 py-1 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:border-info hover:bg-surface-1"
              key={line.id}
            >
              <span className="select-none text-right text-text-subtle">
                {index + 1}
              </span>
              <span className="text-info">{line.level}</span>
              <span className="min-w-0 break-words text-text-secondary">
                {line.message}
              </span>
            </div>
          ))
        ) : (
          <p className="px-2 py-8 text-center text-supporting text-text-tertiary">
            No log output yet.
          </p>
        )}
        {follow ? (
          <span className="mt-2 border-t border-border-subtle px-2 pt-2 text-success">
            Following live output
          </span>
        ) : null}
      </div>
    </ScrollArea>
  )
}
