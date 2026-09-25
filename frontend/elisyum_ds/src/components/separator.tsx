import type { HTMLAttributes } from 'react'

export interface SeparatorProps extends HTMLAttributes<HTMLDivElement> {
  orientation?: 'horizontal' | 'vertical'
}

export function Separator({
  className = '',
  orientation = 'horizontal',
  ...props
}: SeparatorProps) {
  return (
    <div
      {...props}
      aria-orientation={orientation}
      role="separator"
      className={`${orientation === 'vertical' ? 'h-full w-px' : 'h-px w-full'} bg-border-subtle ${className}`}
    />
  )
}
