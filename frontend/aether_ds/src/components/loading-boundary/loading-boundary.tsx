import type { ReactNode } from 'react'
import { Spinner } from '../spinner/spinner'
export interface LoadingBoundaryProps {
  loading: boolean
  children: ReactNode
  fallback?: ReactNode
  variant?: 'page' | 'section' | 'card' | 'table' | 'overlay' | 'action'
}
export function LoadingBoundary({
  children,
  fallback,
  loading,
  variant = 'section',
}: LoadingBoundaryProps) {
  if (!loading) return children
  if (fallback) return fallback
  const sizes = {
    page: 'min-h-64',
    section: 'min-h-32',
    card: 'min-h-24 rounded-xl border border-border bg-surface-card/60 shadow-sm',
    table: 'min-h-48 rounded-xl border border-border bg-surface-card/60 shadow-sm',
    overlay: 'min-h-32 rounded-xl border border-border bg-surface-modal/80 shadow-xl backdrop-blur-xl',
    action: 'min-h-6',
  }
  return (
    <div
      role="status"
      aria-label="Loading"
      className={`flex items-center justify-center p-6 ${sizes[variant]}`}
    >
      <Spinner size={variant === 'action' ? 'sm' : 'md'} />
    </div>
  )
}
