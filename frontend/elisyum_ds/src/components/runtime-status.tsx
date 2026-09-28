import type { ReactNode } from 'react'
import type { BadgeTone } from './badge'
import { Marker } from './marker'

export type RuntimeStatusValue =
  | 'healthy'
  | 'deploying'
  | 'degraded'
  | 'failed'
  | 'stopped'
  | 'unknown'
export interface RuntimeStatusProps {
  status: RuntimeStatusValue
  label?: ReactNode
  className?: string
}
const tones: Record<RuntimeStatusValue, BadgeTone> = {
  healthy: 'success',
  deploying: 'accent',
  degraded: 'warning',
  failed: 'danger',
  stopped: 'neutral',
  unknown: 'neutral',
}
const statusClasses: Record<RuntimeStatusValue, string> = {
  healthy: 'border-success/20 bg-success-soft text-success-strong',
  deploying: 'border-action/20 bg-action-soft text-action-strong',
  degraded: 'border-warning/20 bg-warning-soft text-warning-strong',
  failed: 'border-danger/20 bg-danger-soft text-danger-strong',
  stopped: 'border-current/20 bg-surface-3 text-text-secondary',
  unknown: 'border-current/20 bg-surface-3 text-text-secondary',
}
export function RuntimeStatus({ className = '', label, status }: RuntimeStatusProps) {
  return (
    <div
      aria-label={`Runtime status: ${String(label ?? status)}`}
      className={`inline-flex min-h-6 items-center gap-2 rounded-full border px-2.5 py-1 text-label ${statusClasses[status]} ${className}`}
      role="status"
    >
      <Marker
        aria-hidden="true"
        className={
          status === 'healthy' || status === 'deploying'
            ? 'motion-safe:animate-pulse'
            : ''
        }
        tone={tones[status]}
      />
      <span>{label ?? status}</span>
    </div>
  )
}
