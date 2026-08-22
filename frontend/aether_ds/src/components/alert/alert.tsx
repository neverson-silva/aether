import type { Icon } from '@phosphor-icons/react'
import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const alert = tv({
  base: 'flex items-start gap-3 rounded-lg border p-4',
  variants: {
    tone: {
      info: 'border-status-info/30 bg-status-info-container text-foreground',
      success:
        'border-status-success/30 bg-status-success-container text-foreground',
      warning:
        'border-status-warning/30 bg-status-warning-container text-foreground',
      danger:
        'border-status-danger/30 bg-status-danger-container text-foreground',
    },
  },
  defaultVariants: { tone: 'info' },
})
export interface AlertProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof alert> {
  icon?: Icon
  title?: string
  dismissible?: boolean
  onDismiss?: () => void
}
export function Alert({
  children,
  className = '',
  dismissible,
  icon: IconComponent,
  onDismiss,
  title,
  tone,
  ...props
}: AlertProps) {
  return (
    <div
      role={tone === 'danger' ? 'alert' : 'status'}
      className={alert({ tone, className })}
      {...props}
    >
      {IconComponent ? <IconComponent size={20} aria-hidden="true" /> : null}
      <div className="min-w-0 flex-1 space-y-1">
        {title ? <div className="font-semibold">{title}</div> : null}
        <div className="text-body-sm text-muted-foreground">{children}</div>
      </div>
      {dismissible ? (
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground"
          aria-label="Dismiss"
          onClick={onDismiss}
        >
          ×
        </button>
      ) : null}
    </div>
  )
}
