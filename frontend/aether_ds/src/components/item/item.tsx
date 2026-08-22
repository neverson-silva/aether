import type { HTMLAttributes, ReactNode } from 'react'
export interface ItemProps
  extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  media?: ReactNode
  title: ReactNode
  description?: ReactNode
  metadata?: ReactNode
  actions?: ReactNode
  selected?: boolean
  interactive?: boolean
}
export function Item({
  actions,
  className = '',
  description,
  interactive,
  media,
  metadata,
  selected,
  title,
  ...props
}: ItemProps) {
  return (
    <div
      className={`flex items-center gap-3 rounded-md p-3 ${interactive ? 'cursor-pointer transition-colors hover:bg-surface-container focus-within:ring-2 focus-within:ring-ring' : ''} ${selected ? 'bg-primary/10 text-primary' : ''} ${className}`}
      data-selected={selected || undefined}
      {...props}
    >
      {media ? <div className="shrink-0">{media}</div> : null}
      <div className="min-w-0 flex-1">
        <div className="truncate font-semibold">{title}</div>
        {description ? (
          <div className="truncate text-body-sm text-muted-foreground">
            {description}
          </div>
        ) : null}
      </div>
      {metadata ? (
        <div className="shrink-0 text-body-sm text-muted-foreground">
          {metadata}
        </div>
      ) : null}
      {actions ? <div className="shrink-0">{actions}</div> : null}
    </div>
  )
}
export interface ItemGroupProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}
export function ItemGroup({
  children,
  className = '',
  ...props
}: ItemGroupProps) {
  return (
    <div className={`space-y-1 ${className}`} {...props}>
      {children}
    </div>
  )
}
