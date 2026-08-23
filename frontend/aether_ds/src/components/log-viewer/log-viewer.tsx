import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import { Check, Copy, MagnifyingGlass, Pause, Play } from '@phosphor-icons/react'
import { useEffect, useMemo, useRef, useState } from 'react'
import '@xterm/xterm/css/xterm.css'

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
  fullHeight?: boolean
}

const severityColor: Record<NonNullable<LogLine['severity']>, string> = {
  info: '\u001b[38;5;250m',
  success: '\u001b[38;5;114m',
  warning: '\u001b[38;5;221m',
  error: '\u001b[38;5;203m',
}

const resetColor = '\u001b[0m'
const mutedColor = '\u001b[38;5;244m'

function formatLogLine(line: LogLine, index: number): string {
  const number = `${index + 1}`.padStart(4, ' ')
  const timestamp = line.timestamp ? `${line.timestamp} ` : ''
  const color = severityColor[line.severity ?? 'info']
  return `${mutedColor}${number} ${timestamp}${resetColor}${color}${line.message}${resetColor}\r\n`
}

export function LogViewer({
  disconnected,
  followTail = true,
  lines,
  loading,
  onFollowTailChange,
  fullHeight = false,
}: LogViewerProps) {
  const [query, setQuery] = useState('')
  const [copied, setCopied] = useState(false)
  const [localFollowTail, setLocalFollowTail] = useState(followTail)
  const terminalHostRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<Terminal | null>(null)
  const previousVisibleRef = useRef<LogLine[]>([])
  const previousQueryRef = useRef(query)
  const atBottomRef = useRef(true)
  const followTailRef = useRef(followTail)
  const onFollowTailChangeRef = useRef(onFollowTailChange)
  const isControlled = Boolean(onFollowTailChange)
  const activeFollowTail = isControlled ? followTail : localFollowTail
  const visible = useMemo(
    () =>
      lines.filter((line) =>
        line.message.toLowerCase().includes(query.toLowerCase()),
      ),
    [lines, query],
  )

  useEffect(() => {
    setLocalFollowTail(followTail)
    followTailRef.current = followTail
  }, [followTail])

  useEffect(() => {
    onFollowTailChangeRef.current = onFollowTailChange
  }, [onFollowTailChange])

  useEffect(() => {
    const host = terminalHostRef.current
    if (!host) return
    const terminal = new Terminal({
      convertEol: true,
      cursorBlink: false,
      disableStdin: true,
      fontFamily: '"JetBrains Mono", monospace',
      fontSize: 11,
      lineHeight: 1.35,
      scrollback: 10000,
      theme: {
        background: '#050505',
        foreground: '#d4d4d4',
        selectionBackground: '#264f78',
      },
    })
    const fit = new FitAddon()
    terminal.loadAddon(fit)
    terminal.open(host)
    terminalRef.current = terminal
    const scrollSubscription = terminal.onScroll(() => {
      const buffer = terminal.buffer.active
      const atBottom = buffer.viewportY >= Math.max(0, buffer.baseY - 1)
      if (atBottomRef.current === atBottom) return
      atBottomRef.current = atBottom
      followTailRef.current = atBottom
      setLocalFollowTail(atBottom)
      onFollowTailChangeRef.current?.(atBottom)
    })
    const resize = () => {
      try {
        fit.fit()
      } catch {
        return
      }
    }
    resize()
    const observer = new ResizeObserver(resize)
    observer.observe(host)
    return () => {
      observer.disconnect()
      scrollSubscription.dispose()
      terminal.dispose()
      terminalRef.current = null
    }
  }, [])

  useEffect(() => {
    const terminal = terminalRef.current
    if (!terminal) return
    const previousVisible = previousVisibleRef.current
    const previousQuery = previousQueryRef.current
    const wasAtBottom = activeFollowTail && atBottomRef.current
    const isAppend =
      previousQuery === query &&
      previousVisible.length <= visible.length &&
      previousVisible.every((line, index) => line.id === visible[index]?.id)

    if (loading) {
      terminal.reset()
      terminal.write(`${mutedColor}Loading logs...${resetColor}\r\n`)
      previousVisibleRef.current = []
      previousQueryRef.current = query
      return
    }
    if (!visible.length) {
      terminal.reset()
      terminal.write(`${mutedColor}No matching logs${resetColor}\r\n`)
      previousVisibleRef.current = []
      previousQueryRef.current = query
      return
    }

    if (isAppend) {
      const additions = visible.slice(previousVisible.length)
      if (additions.length) {
        terminal.write(
          additions
            .map((line, index) =>
              formatLogLine(line, previousVisible.length + index),
            )
            .join(''),
          () => {
            if (followTailRef.current && atBottomRef.current) {
              terminal.scrollToBottom()
            }
          },
        )
      }
    } else {
      const previousViewport = terminal.buffer.active.viewportY
      terminal.reset()
      terminal.write(visible.map(formatLogLine).join(''), () => {
        if (wasAtBottom) {
          terminal.scrollToBottom()
        } else {
          terminal.scrollToLine(
            Math.min(previousViewport, terminal.buffer.active.baseY),
          )
        }
      })
    }
    previousVisibleRef.current = visible
    previousQueryRef.current = query
  }, [activeFollowTail, loading, query, visible])

  const updateFollowTail = (next: boolean) => {
    followTailRef.current = next
    setLocalFollowTail(next)
    atBottomRef.current = next
    onFollowTailChangeRef.current?.(next)
    if (next) terminalRef.current?.scrollToBottom()
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
    <section className={`flex min-h-0 flex-col overflow-hidden bg-surface-lowest ${fullHeight ? 'h-full flex-1' : 'rounded-lg border border-border'}`}>
      <header className="flex min-h-12 shrink-0 flex-wrap items-center gap-2 border-b border-border bg-surface-card px-3 py-2">
        <div className="flex min-w-48 max-w-2xl flex-1 items-center gap-2 rounded-md border border-border px-2">
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
          onClick={() => updateFollowTail(!activeFollowTail)}
          aria-pressed={activeFollowTail}
          className="inline-flex shrink-0 items-center gap-1 rounded-md border border-transparent px-2 py-1.5 text-body-sm text-muted-foreground transition-colors hover:border-border hover:bg-surface-container hover:text-foreground active:bg-surface-container-high"
        >
          {activeFollowTail ? <Pause size={16} /> : <Play size={16} />}{' '}
          {activeFollowTail ? 'Following' : 'Paused'}
        </button>
        <button
          type="button"
          onClick={copyVisibleLogs}
          disabled={!visible.length}
          className="inline-flex shrink-0 items-center gap-1 rounded-md border border-transparent px-2 py-1.5 text-body-sm text-muted-foreground transition-colors hover:border-border hover:bg-surface-container hover:text-foreground active:bg-surface-container-high disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copied ? <Check size={16} /> : <Copy size={16} />}{' '}
          {copied ? 'Copied' : 'Copy logs'}
        </button>
        {disconnected ? (
          <span className="text-body-sm text-status-danger">Disconnected</span>
        ) : null}
      </header>
      <div
        ref={terminalHostRef}
        className="min-h-0 w-full flex-1 overflow-hidden bg-surface-lowest p-3"
        style={{
          height: fullHeight ? '100%' : '28rem',
          maxHeight: fullHeight ? undefined : 'calc(100dvh - 14rem)',
        }}
        aria-live="polite"
      />
    </section>
  )
}
