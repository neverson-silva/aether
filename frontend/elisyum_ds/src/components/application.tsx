import type { ReactNode } from 'react'
import { AppHeader } from './app-header'
import { Container } from './layout'

export interface ApplicationProps {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
}
export function Application({
  actions,
  children,
  className = '',
  description,
  title,
}: ApplicationProps) {
  return (
    <div className={`min-h-screen bg-canvas text-text-primary ${className}`}>
      <AppHeader
        actions={actions}
        description={description}
        title={title}
      />
      <main>
        <Container className="py-6 sm:py-8">{children}</Container>
      </main>
    </div>
  )
}
