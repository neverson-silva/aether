import { forwardRef, type HTMLAttributes, type ReactNode } from 'react'

export interface ScrollAreaProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
  orientation?: 'vertical' | 'horizontal' | 'both'
}

export const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(function ScrollArea({
  children,
  className = '',
  orientation = 'vertical',
  ...props
}, ref) {
  const overflow =
    orientation === 'both'
      ? 'overflow-auto'
      : orientation === 'horizontal'
        ? 'overflow-x-auto'
        : 'overflow-y-auto'
  return (
    <div
      {...props}
      className={`${overflow} overscroll-contain focus-visible:outline-2 focus-visible:outline-focus ${className}`}
      ref={ref}
    >
      {children}
    </div>
  )
})
