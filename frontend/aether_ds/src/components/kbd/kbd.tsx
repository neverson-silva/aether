import type { HTMLAttributes } from 'react'
export interface KbdProps extends HTMLAttributes<HTMLElement> {
  keys?: string[]
}
export function Kbd({ children, className = '', keys, ...props }: KbdProps) {
  return (
    <kbd
      className={`inline-flex items-center gap-1 rounded border border-border bg-surface-container px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground shadow-sm ${className}`}
      {...props}
    >
      {keys?.map((key) => <kbd key={key}>{key}</kbd>) ?? children}
    </kbd>
  )
}
