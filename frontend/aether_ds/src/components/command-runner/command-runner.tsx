import { Copy, Play, Stop } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface CommandRunnerProps {
  command?: string
  target?: ReactNode
  output?: string
  status?: 'idle' | 'running' | 'success' | 'error' | 'timeout' | 'cancelled'
  onRun?: (command: string) => void
  onCancel?: () => void
  onRetry?: () => void
  onCopy?: () => void
  permissionDenied?: boolean
}
export function CommandRunner({
  command: initialCommand = '',
  onCancel,
  onCopy,
  onRetry,
  onRun,
  output,
  permissionDenied,
  status = 'idle',
  target,
}: CommandRunnerProps) {
  const [command, setCommand] = useState(initialCommand)
  return (
    <section className="overflow-hidden rounded-lg border border-border bg-surface-card">
      <header className="flex flex-wrap items-center gap-2 border-b border-border p-3">
        <input
          value={command}
          onChange={(event) => setCommand(event.target.value)}
          aria-label="Command"
          placeholder="Enter command"
          className="h-9 min-w-48 flex-1 rounded-md border border-border bg-surface-background px-3 font-mono text-code-md outline-none focus:border-primary"
        />
        {target ? (
          <span className="text-body-sm text-muted-foreground">{target}</span>
        ) : null}
        <button
          type="button"
          disabled={permissionDenied || status === 'running' || !command}
          onClick={() => onRun?.(command)}
          className="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-2 text-body-sm text-primary-foreground disabled:opacity-50"
        >
          <Play size={16} />
          Run
        </button>
        {status === 'running' ? (
          <button
            type="button"
            onClick={onCancel}
            className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-body-sm"
          >
            <Stop size={16} />
            Cancel
          </button>
        ) : null}
      </header>
      {permissionDenied ? (
        <p className="p-4 text-body-sm text-status-danger">
          You do not have permission to run commands on this target.
        </p>
      ) : null}
      <pre className="min-h-32 overflow-auto p-4 font-mono text-code-md text-foreground">
        {output ??
          (status === 'running' ? 'Running...' : 'Output will appear here.')}
      </pre>
      <footer className="flex justify-end gap-3 border-t border-border p-3">
        {status === 'error' || status === 'timeout' ? (
          <button
            type="button"
            onClick={onRetry}
            className="text-body-sm font-semibold text-primary"
          >
            Retry
          </button>
        ) : null}
        <button
          type="button"
          onClick={onCopy}
          className="inline-flex items-center gap-1 text-body-sm text-muted-foreground hover:text-foreground"
        >
          <Copy size={16} />
          Copy output
        </button>
      </footer>
    </section>
  )
}
