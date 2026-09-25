import type { HTMLAttributes, ReactNode } from 'react'

export interface ItemProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  leading?: ReactNode
  trailing?: ReactNode
  title: ReactNode
  description?: ReactNode
  selected?: boolean
}

export interface ItemGroupProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}

export function Item({
  className = '',
  description,
  leading,
  onClick,
  onKeyDown,
  selected = false,
  title,
  trailing,
  ...props
}: ItemProps) {
  const actionable = Boolean(onClick || props.role === 'button')
  return (
    <div
      {...props}
      aria-selected={selected}
      className={`flex min-h-14 items-center gap-3 rounded-sm border-l-2 px-3 py-2 transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${selected ? 'border-action bg-action-soft' : 'border-transparent hover:bg-surface-2'} ${actionable ? 'cursor-pointer focus-visible:outline-2 focus-visible:outline-focus' : ''} ${className}`}
      onClick={onClick}
      onKeyDown={(event) => {
        if (actionable && (event.key === 'Enter' || event.key === ' ')) {
          event.preventDefault()
          onClick?.(event as never)
        }
        onKeyDown?.(event)
      }}
      role={actionable ? 'button' : props.role}
      tabIndex={actionable ? 0 : props.tabIndex}
    >
      <div className="shrink-0 text-text-tertiary">{leading}</div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-body text-text-primary">{title}</div>
        {description ? (
          <div className="truncate text-supporting text-text-tertiary">
            {description}
          </div>
        ) : null}
      </div>
      <div className="shrink-0">{trailing}</div>
    </div>
  )
}

export function ItemGroup({ children, className = '', ...props }: ItemGroupProps) {
  return (
    <div
      {...props}
      className={`grid gap-1 ${className}`}
    >
      {children}
    </div>
  )
}
