import type { ReactNode } from 'react'
import { Alert } from './alert'
export type OfflineState = 'offline' | 'reconnecting' | 'online'
export interface OfflineIndicatorProps {
  state: OfflineState
  children?: ReactNode
  className?: string
}
export function OfflineIndicator({
  children,
  className = '',
  state,
}: OfflineIndicatorProps) {
  if (state === 'online') return null
  return (
    <div
      aria-live="assertive"
      className={`rounded-lg ${className}`}
    >
      <Alert
        title={state === 'offline' ? 'You are offline' : 'Reconnecting'}
        tone={state === 'offline' ? 'danger' : 'warning'}
      >
        {children ?? 'Changes will resume when the connection is restored.'}
      </Alert>
    </div>
  )
}
