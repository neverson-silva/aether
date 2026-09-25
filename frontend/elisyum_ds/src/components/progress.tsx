import type { HTMLAttributes } from 'react'

export interface ProgressProps extends HTMLAttributes<HTMLDivElement> {
  value?: number
  max?: number
  label?: string
}

export function Progress({
  className = '',
  label = 'Progress',
  max = 100,
  value = 0,
  ...props
}: ProgressProps) {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100))
  return (
    <div
      {...props}
      aria-label={label}
      aria-valuemax={max}
      aria-valuemin={0}
      aria-valuenow={value}
      role="progressbar"
      className={`grid gap-2 ${className}`}
    >
      <div className="h-2 overflow-hidden rounded-pill border border-border-subtle bg-surface-3">
        <div
          className="h-full origin-left rounded-pill bg-action transition-transform duration-[var(--ely-duration-standard)] ease-ely-out motion-reduce:transition-none"
          style={{ transform: `scaleX(${percentage / 100})` }}
        />
      </div>
      <span className="sr-only">{Math.round(percentage)}% complete</span>
    </div>
  )
}
