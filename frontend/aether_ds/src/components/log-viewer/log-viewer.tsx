import { Check, Copy, MagnifyingGlass, Pause, Play } from '@phosphor-icons/react'
import { useEffect, useMemo, useRef, useState } from 'react'
export interface LogLine {
  id: string
  timestamp?: string
  severity?: 'info' | 'success' | 'warning' | 'error'
  message: string
}
export interface LogViewerProps {
  lines: LogLine[]
  followTail?: boolean
  onFollowTailChange?: (value: boolean) => void
  loading?: boolean
  disconnected?: boolean
  onCopy?: (line: LogLine) => void
}
export function LogViewer({
  disconnected,
  followTail = true,
  lines,
  loading,
  onCopy,
  onFollowTailChange,
}: LogViewerProps) {
  const [query, setQuery] = useState('')
  const [copied, setCopied] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const visible = useMemo(
    () =>
      lines.filter((line) =>
        line.message.toLowerCase().includes(query.toLowerCase()),
      ),
    [lines, query],
  )
  useEffect(() => {
    if (!followTail || !scrollRef.current) return
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight
  }, [followTail, lines.length])
  const tone = {
    info: 'text-muted-foreground',
    success: 'text-status-success',
    warning: 'text-status-warning',
    error: 'text-status-danger',
  }
  const copyVisibleLogs = async () => {
    try {
      await navigator.clipboard.writeText(visible.map((line) => line.message).join('\n'))
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1400)
    } catch {
      setCopied(false)
    }
  }
  return (
    <section
      className="overflow-hidden rounded-lg border border-border bg-surface-lowest"
      style={{ backgroundColor: 'var(--aether-black)' }}
    >
      <header className="flex flex-wrap items-center gap-2 border-b border-border bg-surface-card p-2">
        <div className="flex min-w-48 flex-1 items-center gap-2 rounded-md border border-border px-2">
          <MagnifyingGlass size={16} className="text-muted-foreground" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search logs"
            className="h-8 min-w-0 flex-1 bg-transparent text-body-sm text-foreground outline-none"
            aria-label="Search logs"
          />
        </div>
        <button
          type="button"
          onClick={() => onFollowTailChange?.(!followTail)}
          className="inline-flex items-center gap-1 rounded-md px-2 py-1.5 text-body-sm text-muted-foreground hover:bg-surface-container"
        >
          {followTail ? <Pause size={16} /> : <Play size={16} />}{' '}
          {followTail ? 'Following' : 'Paused'}
        </button>
        <button
          type="button"
          onClick={copyVisibleLogs}
          disabled={!visible.length}
          className="inline-flex items-center gap-1 rounded-md px-2 py-1.5 text-body-sm text-muted-foreground hover:bg-surface-container disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copied ? <Check size={16} /> : <Copy size={16} />}{' '}
          {copied ? 'Copied' : 'Copy logs'}
        </button>
        {disconnected ? (
          <span className="text-body-sm text-status-danger">Disconnected</span>
        ) : null}
      </header>
      <div
        ref={scrollRef}
        className="h-[28rem] max-h-[calc(100vh-14rem)] min-h-0 overflow-x-hidden overflow-y-auto bg-surface-lowest p-3 font-mono text-[11px] leading-5"
        style={{ backgroundColor: 'var(--aether-black)', height: '28rem', maxHeight: 'calc(100vh - 14rem)', overflowY: 'auto', overflowX: 'hidden' }}
        aria-live="polite"
      >
        {loading ? (
          <div className="text-muted-foreground">Loading logs...</div>
        ) : visible.length ? (
          visible.map((line, index) => (
            <button
              type="button"
              key={line.id}
              onClick={() => onCopy?.(line)}
              className="group flex w-full items-start justify-start gap-3 rounded px-2 py-1 text-left !text-left hover:bg-surface-container"
              style={{ textAlign: 'left' }}
            >
              <span className="w-10 shrink-0 select-none text-end text-muted-foreground/60">
                {index + 1}
              </span>
              {line.timestamp ? (
                <span className="shrink-0 text-left text-muted-foreground/70">
                  {line.timestamp}
                </span>
              ) : null}
              <span
                className={`${tone[line.severity ?? 'info']} min-w-0 flex-1 whitespace-pre-wrap break-words text-left`}
              >
                {line.message}
              </span>
            </button>
          ))
        ) : (
          <div className="p-6 text-center font-sans text-body-sm text-muted-foreground">
            No matching logs
          </div>
        )}
      </div>
    </section>
  )
}
