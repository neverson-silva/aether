import type { CSSProperties, ReactNode } from 'react'
export interface AspectRatioProps {
  ratio?: number
  children: ReactNode
  className?: string
  style?: CSSProperties
}
export function AspectRatio({
  children,
  className = '',
  ratio = 16 / 9,
  style,
}: AspectRatioProps) {
  return (
    <div
      className={`relative w-full overflow-hidden ${className}`}
      style={{ aspectRatio: ratio, ...style }}
    >
      {children}
    </div>
  )
}
