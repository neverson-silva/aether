import type { HTMLAttributes, ReactNode } from 'react'

export interface DirectionProviderProps extends HTMLAttributes<HTMLDivElement> {
  direction?: 'ltr' | 'rtl'
  children: ReactNode
}
export function DirectionProvider({
  children,
  className = '',
  direction = 'ltr',
  ...props
}: DirectionProviderProps) {
  return (
    <div
      {...props}
      dir={direction}
      className={`contents ${className}`}
    >
      {children}
    </div>
  )
}
