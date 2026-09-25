import type { HTMLAttributes, ReactNode } from 'react'

export interface EmptyStateProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
  icon?: ReactNode
}

export function EmptyState({
  action,
  className = '',
  description,
  icon,
  title,
  ...props
}: EmptyStateProps) {
  return (
    <div
      {...props}
      className={`grid justify-items-center gap-3 rounded-xl border border-border-subtle bg-surface-1/80 px-6 py-10 text-center ${className}`}
    >
      {icon ? (
        <div
          aria-hidden="true"
          className="grid size-10 place-items-center rounded-lg bg-surface-2 text-action-strong"
        >
          {icon}
        </div>
      ) : null}
      <h3 className="text-section-title text-text-primary">{title}</h3>
      {description ? (
        <p className="max-w-md text-supporting text-text-tertiary">{description}</p>
      ) : null}
      {action}
    </div>
  )
}
