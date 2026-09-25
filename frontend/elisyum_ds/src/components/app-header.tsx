import type { ReactNode } from 'react'
import { Breadcrumb, type BreadcrumbItem } from './breadcrumb'

export interface AppHeaderProps {
  title: ReactNode
  description?: ReactNode
  breadcrumbs?: BreadcrumbItem[]
  actions?: ReactNode
  className?: string
}

export function AppHeader({
  actions,
  breadcrumbs,
  className = '',
  description,
  title,
}: AppHeaderProps) {
  return (
    <header
      data-material="translucent"
      className={`sticky top-0 z-20 grid gap-3 border-b border-border-subtle bg-surface-0/85 px-4 py-4 backdrop-blur-[18px] transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none sm:px-6 sm:py-5 ${className}`}
    >
      {breadcrumbs ? <Breadcrumb items={breadcrumbs} /> : null}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="grid min-w-0 gap-1">
          <h1 className="text-page-title text-text-primary">{title}</h1>
          {description ? (
            <p className="max-w-2xl text-supporting text-text-tertiary">
              {description}
            </p>
          ) : null}
        </div>
        {actions ? (
          <div className="flex shrink-0 flex-wrap gap-2">{actions}</div>
        ) : null}
      </div>
    </header>
  )
}
