import type { HTMLAttributes } from 'react'

export interface SpinnerProps extends HTMLAttributes<HTMLSpanElement> {
  label?: string
  size?: 'sm' | 'md' | 'lg'
}

export function Spinner({
  className = '',
  label = 'Loading',
  size = 'md',
  ...props
}: SpinnerProps) {
  const sizeClass = size === 'sm' ? 'size-3' : size === 'lg' ? 'size-6' : 'size-4'
  return (
    <span
      {...props}
      aria-label={label}
      role="status"
      className={`inline-block ${sizeClass} motion-safe:animate-spin rounded-full border-2 border-current border-t-transparent text-action ${className}`}
    />
  )
}
