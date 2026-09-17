import { X } from '@phosphor-icons/react'
import type { Icon } from '@phosphor-icons/react'
import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const alert = tv({
  base: 'flex items-start gap-3 rounded-xl border p-4 shadow-sm',
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
          className="shrink-0 rounded-lg p-1 text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.96]"
          aria-label="Dismiss"
          onClick={onDismiss}
        >
          <X size={16} aria-hidden="true" />
        </button>
      ) : null}
    </div>
  )
}
