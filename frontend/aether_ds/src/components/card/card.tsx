import type { HTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const card = tv({
  base: 'rounded-lg border',
  variants: {
    variant: {
      default: 'border-border bg-surface-card',
      interactive:
        'border-border bg-surface-card transition-colors hover:border-primary/50 hover:bg-surface-container',
      selectable:
        'border-border bg-surface-card data-[selected=true]:border-primary data-[selected=true]:bg-primary/10',
      metric: 'border-border bg-surface-card p-5',
      resource: 'border-border bg-surface-card',
      danger: 'border-status-danger/30 bg-status-danger-container/10',
      glass: 'border-glass-border bg-glass-bg backdrop-blur-md',
      elevated: 'border-border bg-surface-card shadow-lg',
    },
    padding: { none: 'p-0', sm: 'p-3', md: 'p-4', lg: 'p-6' },
  },
  defaultVariants: { variant: 'default', padding: 'md' },
})
export interface CardProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof card> {
  selected?: boolean
  as?: 'div' | 'article' | 'section'
  header?: ReactNode
  footer?: ReactNode
}
export function Card({
  as = 'div',
  children,
  className = '',
  footer,
  header,
  padding,
  selected,
  variant,
  ...props
}: CardProps) {
  const Component = as
  return (
    <Component
      className={card({ variant, padding, className })}
      data-selected={selected || undefined}
      {...props}
    >
      {header ? (
        <div className="border-b border-border px-4 py-3">{header}</div>
      ) : null}
      <div>{children}</div>
      {footer ? (
        <div className="border-t border-border px-4 py-3">{footer}</div>
      ) : null}
    </Component>
  )
}
