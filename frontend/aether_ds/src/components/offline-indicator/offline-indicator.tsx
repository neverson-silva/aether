import {
  CloudArrowDown,
  CloudArrowUp,
  CloudCheck,
  CloudSlash,
  Warning,
} from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export type OfflineState =
  | 'offline'
  | 'reconnecting'
  | 'stale'
  | 'queued'
  | 'synced'
  | 'conflict'
export interface OfflineIndicatorProps {
  state: OfflineState
  label?: ReactNode
  queuedCount?: number
  onRetry?: () => void
}
const copy: Record<OfflineState, string> = {
  offline: 'Offline',
  reconnecting: 'Reconnecting',
  stale: 'Data may be stale',
  queued: 'Changes queued',
  synced: 'Synced',
  conflict: 'Sync conflict',
}
export function OfflineIndicator({
  label,
  onRetry,
  queuedCount,
  state,
}: OfflineIndicatorProps) {
  const Icon =
    state === 'offline'
      ? CloudSlash
      : state === 'reconnecting'
        ? CloudArrowUp
        : state === 'synced'
          ? CloudCheck
          : state === 'conflict'
            ? Warning
            : CloudArrowDown
  const tone =
    state === 'synced'
      ? 'text-status-success'
      : state === 'conflict' || state === 'offline'
        ? 'text-status-danger'
        : state === 'stale' || state === 'queued'
          ? 'text-status-warning'
          : 'text-muted-foreground'
  return (
    <div
      role="status"
      className={`inline-flex items-center gap-2 text-body-sm ${tone}`}
    >
      <Icon
        size={16}
        className={state === 'reconnecting' ? 'animate-pulse' : ''}
        aria-hidden="true"
      />
      <span>
        {label ?? copy[state]}
        {state === 'queued' && queuedCount ? ` (${queuedCount})` : ''}
      </span>
      {onRetry && (state === 'offline' || state === 'conflict') ? (
        <button
          type="button"
          onClick={onRetry}
          className="font-semibold underline underline-offset-2"
        >
          Retry
        </button>
      ) : null}
    </div>
  )
}
