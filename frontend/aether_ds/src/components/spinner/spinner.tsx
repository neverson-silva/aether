import type { HTMLAttributes } from 'react'
export interface SpinnerProps extends HTMLAttributes<HTMLSpanElement> {
  size?: 'sm' | 'md' | 'lg'
  label?: string
}
export function Spinner({
  className = '',
  label: accessibleLabel,
  size = 'md',
  ...props
}: SpinnerProps) {
  const sizes = { sm: 'size-3', md: 'size-5', lg: 'size-8' }
  return (
    <span
      role={accessibleLabel ? 'status' : undefined}
      className={`inline-block animate-spin rounded-full border-2 border-current border-t-transparent ${sizes[size]} ${className}`}
      {...props}
    >
      {accessibleLabel ? (
        <span className="sr-only">{accessibleLabel}</span>
      ) : null}
    </span>
  )
}
