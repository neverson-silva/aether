import { CheckCircle, Info, Warning, X, XCircle } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
export interface BannerProps {
  title: ReactNode
  description?: ReactNode
  tone?: 'info' | 'success' | 'warning' | 'error'
  action?: ReactNode
  dismissible?: boolean
  onDismiss?: () => void
}
export function Banner({
  action,
  description,
  dismissible,
  onDismiss,
  title,
  tone = 'info',
}: BannerProps) {
  const Icon =
    tone === 'success'
      ? CheckCircle
      : tone === 'warning'
        ? Warning
        : tone === 'error'
          ? XCircle
          : Info
  const styles = {
    info: 'border-status-info/30 bg-status-info-container/20',
    success: 'border-status-success/30 bg-status-success-container/20',
    warning: 'border-status-warning/30 bg-status-warning-container/20',
    error: 'border-status-danger/30 bg-status-danger-container/20',
  }
  return (
    <aside
      role="status"
      className={`flex gap-3 rounded-lg border p-4 text-foreground ${styles[tone]}`}
    >
      <Icon size={20} className="mt-0.5 shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <div className="font-semibold">{title}</div>
        {description ? (
          <div className="mt-1 text-body-sm text-muted-foreground">
            {description}
          </div>
        ) : null}
        {action ? <div className="mt-3">{action}</div> : null}
      </div>
      {dismissible ? (
        <button
          type="button"
          onClick={onDismiss}
          aria-label="Dismiss banner"
          className="shrink-0 text-muted-foreground hover:text-foreground"
        >
          <X size={16} />
        </button>
      ) : null}
    </aside>
  )
}
