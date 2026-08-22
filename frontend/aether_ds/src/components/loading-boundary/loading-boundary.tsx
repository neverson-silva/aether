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
    card: 'min-h-24 rounded-lg border border-border',
    table: 'min-h-48 rounded-lg border border-border',
    overlay: 'min-h-32 rounded-lg bg-surface-modal/80',
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
