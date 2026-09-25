import type { ReactNode } from 'react'
export interface ButtonGroupProps {
  children: ReactNode
  className?: string
}
export function ButtonGroup({ children, className = '' }: ButtonGroupProps) {
  return (
    <fieldset
      aria-label="Actions"
      className={`inline-flex items-center gap-1 border-y border-border-subtle py-1 ${className}`}
    >
      {children}
    </fieldset>
  )
}
