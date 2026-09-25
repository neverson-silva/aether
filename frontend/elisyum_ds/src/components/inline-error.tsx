import type { HTMLAttributes, ReactNode } from 'react'

export interface InlineErrorProps extends HTMLAttributes<HTMLParagraphElement> {
  children: ReactNode
}

export function InlineError({ children, className = '', ...props }: InlineErrorProps) {
  return (
    <p
      {...props}
      aria-live="assertive"
      role="alert"
      className={`text-supporting text-danger ${className}`}
    >
      {children}
    </p>
  )
}
