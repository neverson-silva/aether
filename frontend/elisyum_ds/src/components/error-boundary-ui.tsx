import type { ReactNode } from 'react'
import { Button } from './button'
export interface ErrorBoundaryUIProps {
  title?: ReactNode
  description?: ReactNode
  onRetry?: () => void
  className?: string
}
export function ErrorBoundaryUI({
  className = '',
  description = 'The interface could not load this section.',
  onRetry,
  title = 'Something went wrong',
}: ErrorBoundaryUIProps) {
  return (
    <div
      className={`grid justify-items-start gap-3 rounded-lg border border-l-2 border-danger bg-danger-soft px-5 py-4 ${className}`}
      role="alert"
    >
      <h2 className="text-section-title text-text-primary">{title}</h2>
      <p className="text-supporting text-text-secondary">{description}</p>
      {onRetry ? (
        <Button
          onClick={onRetry}
          tone="danger"
        >
          Try again
        </Button>
      ) : null}
    </div>
  )
}
