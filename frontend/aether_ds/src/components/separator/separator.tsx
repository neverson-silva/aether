import type { HTMLAttributes } from 'react'
export interface SeparatorProps extends HTMLAttributes<HTMLHRElement> {
  orientation?: 'horizontal' | 'vertical'
  decorative?: boolean
}
export function Separator({
  className = '',
  decorative = true,
  orientation = 'horizontal',
  ...props
}: SeparatorProps) {
  return (
    <hr
      role={decorative ? 'presentation' : 'separator'}
      className={`${orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px'} border-0 bg-border ${className}`}
      {...props}
    />
  )
}
