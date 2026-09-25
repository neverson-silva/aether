import { useId, type ReactNode } from 'react'
export interface FormsProps {
  title?: ReactNode
  children: ReactNode
  className?: string
}
export function Forms({
  children,
  className = '',
  title = 'Form patterns',
}: FormsProps) {
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
