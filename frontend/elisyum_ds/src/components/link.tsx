import type { AnchorHTMLAttributes, ReactNode } from 'react'

export interface LinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  children: ReactNode
}

export function Link({ children, className = '', ...props }: LinkProps) {
  return (
    <a
      {...props}
      className={`cursor-pointer font-medium text-action-strong underline decoration-action/40 underline-offset-4 transition-colors duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:text-action focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus ${className}`}
    >
      {children}
    </a>
  )
}
