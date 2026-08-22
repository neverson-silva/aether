import type { Icon } from '@phosphor-icons/react'
import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const badge = tv({
  base: 'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 font-semibold',
  variants: {
    tone: {
      neutral: 'border-border bg-surface-container text-muted-foreground',
      info: 'border-status-info/30 bg-status-info-container text-status-info',
      success:
        'border-status-success/30 bg-status-success-container text-status-success',
      warning:
        'border-status-warning/30 bg-status-warning-container text-status-warning',
      danger:
        'border-status-danger/30 bg-status-danger-container text-status-danger',
      accent: 'border-secondary/30 bg-secondary-container text-secondary',
    },
    size: { sm: 'text-[11px]', md: 'text-body-sm' },
  },
  defaultVariants: { tone: 'neutral', size: 'sm' },
})
export interface BadgeProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badge> {
  icon?: Icon
  dot?: boolean
  onRemove?: () => void
  live?: boolean
}
export function Badge({
  className = '',
  children,
  icon: IconComponent,
  dot,
  live,
  onRemove,
  tone,
  size,
  ...props
}: BadgeProps) {
  return (
    <span
      className={badge({ tone, size, className })}
      aria-live={live ? 'polite' : undefined}
      {...props}
    >
      {dot ? (
        <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
      ) : null}
      {IconComponent ? <IconComponent size={14} aria-hidden="true" /> : null}
      {children}
      {onRemove ? (
        <button
          type="button"
          className="ml-0.5 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={`Remove ${children}`}
          onClick={onRemove}
        >
          ×
        </button>
      ) : null}
    </span>
  )
}
