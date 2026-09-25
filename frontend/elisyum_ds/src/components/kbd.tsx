import type { HTMLAttributes, ReactNode } from 'react'

export interface KbdProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode
}

export function Kbd({ children, className = '', ...props }: KbdProps) {
  return (
    <kbd
      {...props}
      className={`rounded border border-border-default bg-surface-2 px-1.5 py-0.5 font-technical text-label text-text-secondary ${className}`}
    >
      {children}
    </kbd>
  )
}
