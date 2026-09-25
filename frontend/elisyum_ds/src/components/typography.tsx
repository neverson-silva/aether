import type { ElementType, HTMLAttributes, ReactNode } from 'react'

export type TypographyRole =
  | 'display'
  | 'page-title'
  | 'section-title'
  | 'body'
  | 'supporting'
  | 'label'
  | 'metric'
  | 'code'
  | 'log'

export interface TypographyProps extends HTMLAttributes<HTMLElement> {
  role?: TypographyRole
  as?: ElementType
  children: ReactNode
}

const roleClasses: Record<TypographyRole, string> = {
  display: 'text-display text-text-primary',
  'page-title': 'text-page-title text-text-primary',
  'section-title': 'text-section-title text-text-primary',
  body: 'text-body text-text-secondary',
  supporting: 'text-supporting text-text-tertiary',
  label: 'text-label text-text-secondary',
  metric: 'text-metric text-text-primary',
  code: 'font-technical text-code text-text-secondary',
  log: 'font-technical text-log text-text-secondary',
}

export function Typography({
  as: Element = 'p',
  children,
  className = '',
  role = 'body',
  ...props
}: TypographyProps) {
  return (
    <Element
      {...props}
      className={`${roleClasses[role]} ${className}`}
    >
      {children}
    </Element>
  )
}
