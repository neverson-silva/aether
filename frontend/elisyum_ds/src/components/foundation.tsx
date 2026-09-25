import { useId, type ReactNode } from 'react'
export interface FoundationProps {
  title?: ReactNode
  children: ReactNode
  className?: string
}
export function Foundation({
  children,
  className = '',
  title = 'Foundations',
}: FoundationProps) {
  const titleId = useId()
  return (
    <section
      aria-labelledby={titleId}
      className={`grid gap-4 ${className}`}
    >
      <h2
        className="text-section-title text-text-primary"
        id={titleId}
      >
        {title}
      </h2>
      {children}
    </section>
  )
}
