import { CheckCircle, Info, Warning, XCircle } from '@phosphor-icons/react'
import type { HTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const alertVariants = tv({
  base: 'grid gap-1 rounded-lg border-l-2 bg-surface-2 px-4 py-3',
  variants: {
    tone: {
      info: 'border-info bg-info/10 text-text-primary',
      success: 'border-success bg-success-soft text-text-primary',
      warning: 'border-warning bg-warning-soft text-text-primary',
      danger: 'border-danger bg-danger-soft text-text-primary',
    },
  },
  defaultVariants: { tone: 'info' },
})

export interface AlertProps
  extends Omit<HTMLAttributes<HTMLDivElement>, 'title'>,
    VariantProps<typeof alertVariants> {
  title: ReactNode
  children?: ReactNode
}

export function Alert({
  children,
  className = '',
  title,
  tone = 'info',
  ...props
}: AlertProps) {
  const Icon = { danger: XCircle, info: Info, success: CheckCircle, warning: Warning }[
    tone
  ]
  return (
    <div
      {...props}
      role={tone === 'danger' ? 'alert' : 'status'}
      className={alertVariants({ className, tone })}
    >
      <div className="flex items-center gap-2">
        <Icon
          aria-hidden="true"
          size={18}
          weight="bold"
        />
        <strong className="text-label">{title}</strong>
      </div>
      {children ? (
        <div className="pl-6 text-supporting text-text-secondary">{children}</div>
      ) : null}
    </div>
  )
}
