import type { HTMLAttributes } from 'react'
export interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  variant?: 'text' | 'avatar' | 'card' | 'table'
}
export function Skeleton({
  className = '',
  variant = 'text',
  ...props
}: SkeletonProps) {
  const shapes = {
    text: 'h-4 w-full rounded',
    avatar: 'size-10 rounded-full',
    card: 'h-32 w-full rounded-lg',
    table: 'h-10 w-full rounded',
  }
  return (
    <div
      className={`aether-shimmer bg-surface-container ${shapes[variant]} ${className}`}
      aria-hidden="true"
      {...props}
    />
  )
}
