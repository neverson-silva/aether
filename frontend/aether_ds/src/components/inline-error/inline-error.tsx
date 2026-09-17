import { WarningCircle } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface InlineErrorProps {
  message: ReactNode
  title?: ReactNode
  retryLabel?: string
  onRetry?: () => void
  requestId?: string
  supportAction?: ReactNode
  size?: 'sm' | 'md'
}
export function InlineError({
  message,
  onRetry,
  requestId,
  retryLabel = 'Try again',
  size = 'md',
  supportAction,
  title,
}: InlineErrorProps) {
  return (
    <div
      role="alert"
      className={`flex items-start gap-3 rounded-xl border border-status-danger/30 bg-status-danger-container/15 text-status-danger shadow-sm ${size === 'sm' ? 'p-2 text-body-sm' : 'p-4 text-body-md'}`}
    >
      <WarningCircle
        size={size === 'sm' ? 16 : 20}
        className="mt-0.5 shrink-0"
        aria-hidden="true"
      />
      <div className="min-w-0 flex-1">
        {title ? <div className="font-semibold">{title}</div> : null}
        <div className="mt-1">{message}</div>
        {requestId ? (
          <code className="mt-2 block text-code-md text-muted-foreground">
            Request ID: {requestId}
          </code>
        ) : null}
        <div className="mt-3 flex flex-wrap gap-3">
          {onRetry ? (
            <button
              type="button"
              onClick={onRetry}
              className="rounded-lg px-2 py-1 font-semibold text-primary underline underline-offset-2 outline-none transition-[background-color,transform] duration-150 hover:bg-primary/10 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
            >
              {retryLabel}
            </button>
          ) : null}
          {supportAction}
        </div>
      </div>
    </div>
  )
}
