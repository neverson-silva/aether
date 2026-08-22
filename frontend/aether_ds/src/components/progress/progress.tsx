import type { HTMLAttributes } from 'react'
import { tv, type VariantProps } from 'tailwind-variants'

const progress = tv({
  base: 'w-full overflow-hidden rounded-full bg-surface-container',
  variants: { size: { sm: 'h-1', md: 'h-2', lg: 'h-3' } },
  defaultVariants: { size: 'md' },
})
export interface ProgressProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof progress> {
  value?: number
  max?: number
  label?: string
  status?: 'default' | 'success' | 'warning' | 'danger'
  indeterminate?: boolean
}
export function Progress({
  className = '',
  indeterminate,
  label,
  max = 100,
  size,
  status = 'default',
  value = 0,
  ...props
}: ProgressProps) {
  const colors = {
    default: 'bg-primary',
    success: 'bg-status-success',
    warning: 'bg-status-warning',
    danger: 'bg-status-danger',
  }
  const normalized = Math.min(100, Math.max(0, (value / max) * 100))
  return (
    <div className={className} {...props}>
      <div className={progress({ size })}>
        <div
          className={`h-full rounded-full ${colors[status]} ${indeterminate ? 'w-2/5 animate-[progress_1.4s_ease-in-out_infinite]' : 'transition-[width] duration-300'}`}
          style={{ width: indeterminate ? undefined : `${normalized}%` }}
          role="progressbar"
          aria-label={label}
          aria-valuenow={indeterminate ? undefined : value}
          aria-valuemin={0}
          aria-valuemax={max}
        />
      </div>
    </div>
  )
}
