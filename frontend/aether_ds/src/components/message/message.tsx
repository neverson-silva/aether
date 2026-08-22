import { CheckCircle, Info, Warning, XCircle } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import { tv } from 'tailwind-variants'

export interface MessageProps {
  tone?: 'info' | 'success' | 'warning' | 'error'
  title?: ReactNode
  children?: ReactNode
  action?: ReactNode
  icon?: ReactNode
  className?: string
}
const message = tv({
  base: 'flex gap-3 rounded-lg border p-4',
  variants: {
    tone: {
      info: 'border-status-info/30 bg-status-info-container/20',
      success: 'border-status-success/30 bg-status-success-container/20',
      warning: 'border-status-warning/30 bg-status-warning-container/20',
      error: 'border-status-danger/30 bg-status-danger-container/20',
    },
  },
  defaultVariants: { tone: 'info' },
})
const icons = {
  info: Info,
  success: CheckCircle,
  warning: Warning,
  error: XCircle,
}
export function Message({
  action,
  children,
  className,
  icon,
  title,
  tone = 'info',
}: MessageProps) {
  const Icon = icons[tone]
  return (
    <section
      role={tone === 'error' ? 'alert' : 'status'}
      className={message({ tone, className })}
    >
      <span className="mt-0.5 shrink-0 text-foreground">
        {icon ?? <Icon size={20} aria-hidden="true" />}
      </span>
      <div className="min-w-0 flex-1">
        {title ? (
          <h3 className="font-semibold text-foreground">{title}</h3>
        ) : null}
        {children ? (
          <div className="mt-1 text-body-sm text-muted-foreground">
            {children}
          </div>
        ) : null}
        {action ? <div className="mt-3">{action}</div> : null}
      </div>
    </section>
  )
}
