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
export function RuntimeStatus({ className = '', label, status }: RuntimeStatusProps) {
  return (
    <div
      aria-label={`Runtime status: ${String(label ?? status)}`}
      className={`inline-flex items-center gap-2 ${className}`}
      role="status"
    >
      <Marker
        aria-hidden="true"
        className={status === 'deploying' ? 'motion-safe:animate-pulse' : ''}
        tone={tones[status]}
      />
      <span className="text-supporting text-text-secondary">{label ?? status}</span>
    </div>
  )
}
