import type { HTMLAttributes, ReactNode } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const markerVariants = tv({
  base: 'inline-flex size-2.5 rounded-full',
  variants: {
    tone: {
      neutral: 'bg-neutral-state',
      accent: 'bg-action',
      success: 'bg-success',
      warning: 'bg-warning',
      danger: 'bg-danger',
    },
  },
  defaultVariants: { tone: 'neutral' },
})
export interface MarkerProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof markerVariants> {
  label?: ReactNode
}
export function Marker({
  className = '',
  label,
  tone = 'neutral',
  ...props
}: MarkerProps) {
  return (
    <span
      {...props}
      aria-label={label ? String(label) : undefined}
      className={`${markerVariants({ tone })} ${className}`}
      role={label ? 'img' : undefined}
    />
  )
}
