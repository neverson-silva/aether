import type { HTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const badgeVariants = tv({
  base: 'inline-flex min-h-6 items-center rounded-full border border-current/20 px-2 text-label transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none',
  variants: {
    tone: {
      neutral: 'bg-surface-3 text-text-secondary',
      accent: 'bg-action-soft text-action-strong',
      success: 'bg-success-soft text-success-strong',
      warning: 'bg-warning-soft text-warning-strong',
      danger: 'bg-danger-soft text-danger-strong',
    },
  },
  defaultVariants: {
    tone: 'neutral',
  },
})

export type BadgeTone = NonNullable<VariantProps<typeof badgeVariants>['tone']>

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone
  children: ReactNode
}

export function Badge({
  className = '',
  tone = 'neutral',
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      {...props}
      className={badgeVariants({ className, tone })}
    >
      {children}
    </span>
  )
}
