import type { ReactNode } from 'react'
import { Breadcrumb, type BreadcrumbItem } from '../breadcrumb/breadcrumb'
export interface AppHeaderProps {
  breadcrumb?: BreadcrumbItem[]
  workspace?: ReactNode
  environment?: ReactNode
  search?: ReactNode
  command?: ReactNode
  notifications?: ReactNode
  theme?: ReactNode
  user?: ReactNode
  onNavigate?: (href: string) => void
}
export function AppHeader({
  breadcrumb,
  command,
  environment,
  notifications,
  search,
  theme,
  user,
  workspace,
  onNavigate,
}: AppHeaderProps) {
  return (
    <header className="flex min-h-16 items-center gap-4 border-b border-border bg-surface-background px-4 md:px-6">
      <div className="min-w-0 flex-1 space-y-1">
        {workspace ? <div className="font-semibold">{workspace}</div> : null}
        {breadcrumb ? <Breadcrumb items={breadcrumb} onNavigate={onNavigate} /> : null}
      </div>
      {environment}
      {search}
      {command}
      {notifications}
      {theme}
      {user}
    </header>
  )
}
