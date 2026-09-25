import type { ReactNode } from 'react'
import { Skeleton } from './skeleton'
export interface LoadingBoundaryProps {
  loading: boolean
  children: ReactNode
  fallback?: ReactNode
}
export function LoadingBoundary({
  children,
  fallback = <Skeleton className="h-32 w-full" />,
  loading,
}: LoadingBoundaryProps) {
  return loading ? (
    <div
      aria-busy="true"
      aria-live="polite"
      role="status"
    >
      {fallback}
    </div>
  ) : (
    children
  )
}
