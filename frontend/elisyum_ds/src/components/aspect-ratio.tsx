import type { HTMLAttributes, ReactNode } from 'react'

export interface AspectRatioProps extends HTMLAttributes<HTMLDivElement> {
  ratio?: number
  children: ReactNode
}

export function AspectRatio({
  children,
  className = '',
  ratio = 16 / 9,
  style,
  ...props
}: AspectRatioProps) {
  return (
    <div
      {...props}
      className={`relative overflow-hidden ${className}`}
      style={{ aspectRatio: ratio, ...style }}
    >
      {children}
    </div>
  )
}
