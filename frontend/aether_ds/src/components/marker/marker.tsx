import type { HTMLAttributes, ReactNode } from 'react'
import { tv } from 'tailwind-variants'
export interface MarkerProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode
  tone?: 'neutral' | 'info' | 'success' | 'warning' | 'danger'
  size?: 'sm' | 'md'
}
const marker = tv({
  base: 'inline-flex items-center gap-1.5 rounded-full border font-mono',
  variants: {
    tone: {
      neutral: 'border-border bg-surface-container text-muted-foreground',
      info: 'border-status-info/40 bg-status-info-container/30 text-status-info-foreground',
      success:
        'border-status-success/40 bg-status-success-container/30 text-status-success-foreground',
      warning:
        'border-status-warning/40 bg-status-warning-container/30 text-status-warning-foreground',
      danger:
        'border-status-danger/40 bg-status-danger-container/30 text-status-danger-foreground',
    },
    size: { sm: 'px-2 py-0.5 text-label-caps', md: 'px-2.5 py-1 text-body-sm' },
  },
  defaultVariants: { tone: 'neutral', size: 'sm' },
})
export function Marker({
  children,
  className,
  size = 'sm',
  tone = 'neutral',
  ...props
}: MarkerProps) {
  return (
    <span className={marker({ tone, size, className })} {...props}>
      {children}
    </span>
  )
}
