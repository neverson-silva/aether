import type { Icon } from '@phosphor-icons/react'
import type { HTMLAttributes, ReactNode } from 'react'
export interface EmptyStateProps extends HTMLAttributes<HTMLDivElement> {
  icon?: Icon
  title: string
  description?: string
  action?: ReactNode
  secondaryAction?: ReactNode
}
export function EmptyState({
  action,
  children,
  className = '',
  description,
  icon: IconComponent,
  secondaryAction,
  title,
  ...props
}: EmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border p-10 text-center ${className}`}
      {...props}
    >
      {IconComponent ? (
        <IconComponent
          size={32}
          weight="duotone"
          className="text-muted-foreground"
          aria-hidden="true"
        />
      ) : null}
      <div className="space-y-1">
        <h3 className="font-semibold text-foreground">{title}</h3>
        {description ? (
          <p className="max-w-md text-body-sm text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      {children ? <div>{children}</div> : null}
      <div className="flex gap-2">
        {action}
        {secondaryAction}
      </div>
    </div>
  )
}
