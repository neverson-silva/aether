import type { AnchorHTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const link = tv({
  base: 'inline-flex items-center gap-1 font-semibold underline-offset-4 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
  variants: {
    tone: {
      default: 'text-primary hover:underline',
      muted: 'text-muted-foreground hover:text-foreground hover:underline',
    },
    underline: { true: 'underline', false: 'no-underline' },
    disabled: { true: 'pointer-events-none opacity-50', false: '' },
  },
  defaultVariants: { tone: 'default', underline: false, disabled: false },
})
export interface LinkProps
  extends AnchorHTMLAttributes<HTMLAnchorElement>,
    VariantProps<typeof link> {
  external?: boolean
}
export function Link({
  className = '',
  children,
  disabled,
  external,
  href,
  tone,
  underline,
  ...props
}: LinkProps) {
  return (
    <a
      className={link({ tone, underline, disabled, className })}
      href={disabled ? undefined : href}
      aria-disabled={disabled || undefined}
      target={external ? '_blank' : props.target}
      rel={external ? 'noreferrer' : props.rel}
      {...props}
    >
      {children}
      {external ? <span aria-hidden="true">↗</span> : null}
    </a>
  )
}
