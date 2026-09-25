import type { HTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const cardVariants = tv({
  base: 'rounded-xl border border-border-subtle bg-surface-1',
  variants: {
    padding: { none: 'p-0', sm: 'p-3', md: 'p-4', lg: 'p-6' },
    interactive: {
      true: 'cursor-pointer transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:border-border-default hover:bg-surface-2 focus-within:border-focus',
    },
    actionable: { true: 'cursor-pointer' },
  },
  compoundVariants: [],
  defaultVariants: { actionable: false, padding: 'md', interactive: false },
})

export interface CardProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {
  children: ReactNode
}

export function Card({
  children,
  className = '',
  interactive = false,
  padding = 'md',
  ...props
}: CardProps) {
  const actionable = Boolean(interactive || props.onClick || props.role === 'button')
  return (
    <div
      {...props}
      className={cardVariants({ actionable, className, interactive, padding })}
    >
      {children}
    </div>
  )
}
